---
description: Debug 66 Python implementation. Verbose step-trace debugging using logging module. Includes extensions for PyTorch, TensorFlow, and Google Cloud Storage.
globs: ["*.py", "*.pynb"]
alwaysApply: false
---

# Debug 66 - Python Implementation

## Logging Setup

```python
import logging
import time
import functools
from typing import Any
from pprint import pformat

# Configure D66 logger
d66_logger = logging.getLogger("D66")
d66_logger.setLevel(logging.DEBUG)
if not d66_logger.handlers:
    handler = logging.StreamHandler()
    handler.setFormatter(logging.Formatter("[D66] %(message)s"))
    d66_logger.addHandler(handler)

def d66_log(msg: str, indent: int = 0):
    """Log with optional indentation."""
    d66_logger.debug("  " * indent + msg)

def d66_state(obj: Any, name: str, indent: int = 0):
    """Log object state summary."""
    prefix = "  " * indent
    type_name = type(obj).__name__
    
    if hasattr(obj, 'shape'):  # numpy/pandas/tensor
        d66_logger.debug(f"{prefix}DATA: {name} | {type_name} | shape={obj.shape}")
    elif hasattr(obj, '__len__'):
        d66_logger.debug(f"{prefix}DATA: {name} | {type_name} | len={len(obj)}")
    else:
        d66_logger.debug(f"{prefix}DATA: {name} | {type_name}")
```

## Instrumentation Patterns

### Function Decorator (Recommended)

```python
def d66_trace(func):
    """Decorator for automatic entry/exit logging."""
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        # [D66:START]
        d66_log(f"─── ENTER {func.__name__} ───────────────────")
        for i, arg in enumerate(args):
            d66_log(f"  ARG[{i}]: {type(arg).__name__} = {_summarize(arg)}")
        for k, v in kwargs.items():
            d66_log(f"  KWARG {k}: {type(v).__name__} = {_summarize(v)}")
        start = time.perf_counter()
        # [D66:END]
        
        try:
            result = func(*args, **kwargs)
            
            # [D66:START]
            duration = time.perf_counter() - start
            d66_log(f"─── EXIT {func.__name__} ({duration:.3f}s) → {_summarize(result)} ───")
            # [D66:END]
            return result
            
        except Exception as e:
            # [D66:START]
            d66_log(f"  ERROR in {func.__name__}: {type(e).__name__}: {e}")
            # [D66:END]
            raise
    return wrapper

def _summarize(obj: Any, max_len: int = 100) -> str:
    """Create a brief summary of an object."""
    if obj is None:
        return "None"
    if hasattr(obj, 'shape'):
        return f"{type(obj).__name__}{list(obj.shape)}"
    if isinstance(obj, (list, tuple)):
        return f"{type(obj).__name__}[{len(obj)}]"
    if isinstance(obj, dict):
        return f"dict{{{len(obj)} keys}}"
    if isinstance(obj, str):
        return f"str[{len(obj)}]" if len(obj) > 50 else repr(obj)
    return str(obj)[:max_len]
```

### Manual Function Entry/Exit

```python
def my_function(arg1, arg2):
    # [D66:START] ─────────────────────────
    d66_log("─── ENTER my_function ───────────────────")
    d66_log(f"  ARG: arg1 = {type(arg1).__name__}: {_summarize(arg1)}")
    d66_log(f"  ARG: arg2 = {type(arg2).__name__}: {_summarize(arg2)}")
    _d66_start = time.perf_counter()
    # [D66:END] ───────────────────────────
    
    # ... function body ...
    result = do_work()
    
    # [D66:START]
    d66_log(f"─── EXIT my_function ({time.perf_counter() - _d66_start:.3f}s) → {_summarize(result)} ───")
    # [D66:END]
    return result
```

### DataFrame State Inspection

```python
def d66_df_state(df, name: str, indent: int = 0):
    """Log pandas DataFrame state."""
    # [D66:START]
    d66_log(f"DATA: {name}", indent)
    d66_log(f"  shape: {df.shape}", indent)
    d66_log(f"  dtypes: {dict(df.dtypes)}", indent)
    d66_log(f"  nulls: {df.isnull().sum().to_dict()}", indent)
    d66_log(f"  memory: {df.memory_usage(deep=True).sum() / 1024:.1f} KB", indent)
    if len(df) > 0:
        d66_log(f"  head:\n{df.head(3).to_string()}", indent)
    # [D66:END]
```

### Loop/Iteration Instrumentation

```python
# [D66:START]
_d66_total = len(items)
_d66_interval = max(1, _d66_total // 10)  # Log every 10%
# [D66:END]

for i, item in enumerate(items):
    # [D66:START]
    if i == 0 or i == _d66_total - 1 or i % _d66_interval == 0:
        d66_log(f"  ITER: [{i+1}/{_d66_total}] processing: {_summarize(item)}")
    # [D66:END]
    
    result = process(item)
    
    # [D66:START]
    if i == 0:  # Log first result structure
        d66_log(f"  ITER: first result = {_summarize(result)}")
    # [D66:END]
```

### Conditional Logic

```python
if condition:
    # [D66:START]
    d66_log(f"  BRANCH: condition → TRUE ({condition=})")
    # [D66:END]
    # ... true branch ...
else:
    # [D66:START]
    d66_log(f"  BRANCH: condition → FALSE")
    # [D66:END]
    # ... false branch ...
```

### Error Handling (Hybrid)

```python
# [D66:START]
try:
    # [D66:END]
    
    # ... original risky code ...
    result = risky_operation(data)
    
    # [D66:START]
except Exception as e:
    d66_log(f"  ERROR: {type(e).__name__}: {e}")
    d66_log(f"  ERROR STATE: data = {_summarize(data)}")
    d66_log(f"  ERROR STATE: locals = {list(locals().keys())}")
    raise  # Re-raise with original traceback
# [D66:END]
```

### Class Method Instrumentation

```python
class MyClass:
    def process(self, data):
        # [D66:START] ─────────────────────────
        d66_log(f"─── ENTER {self.__class__.__name__}.process ───")
        d66_log(f"  ARG: data = {_summarize(data)}")
        d66_log(f"  STATE: self.config = {self.config}")
        _d66_start = time.perf_counter()
        # [D66:END] ───────────────────────────
        
        # ... method body ...
        
        # [D66:START]
        d66_log(f"─── EXIT {self.__class__.__name__}.process ({time.perf_counter() - _d66_start:.3f}s) ───")
        # [D66:END]
```

### Async Function Instrumentation

```python
async def my_async_function(arg):
    # [D66:START]
    d66_log(f"─── ENTER (async) my_async_function ───")
    d66_log(f"  ARG: arg = {_summarize(arg)}")
    _d66_start = time.perf_counter()
    # [D66:END]
    
    result = await some_async_operation()
    
    # [D66:START]
    d66_log(f"─── EXIT (async) my_async_function ({time.perf_counter() - _d66_start:.3f}s) ───")
    # [D66:END]
    return result
```

---

## PyTorch Extensions

### Tensor State Inspection

```python
def d66_tensor(t, name: str, indent: int = 0):
    """Log PyTorch tensor state."""
    # [D66:START]
    d66_log(f"TENSOR: {name}", indent)
    d66_log(f"  shape: {list(t.shape)}", indent)
    d66_log(f"  dtype: {t.dtype}", indent)
    d66_log(f"  device: {t.device}", indent)
    d66_log(f"  requires_grad: {t.requires_grad}", indent)
    if t.numel() > 0 and t.numel() <= 1000:
        d66_log(f"  range: [{t.min().item():.4f}, {t.max().item():.4f}]", indent)
        d66_log(f"  mean: {t.float().mean().item():.4f}", indent)
    if t.requires_grad and t.grad is not None:
        d66_log(f"  grad_shape: {list(t.grad.shape)}", indent)
        d66_log(f"  grad_norm: {t.grad.norm().item():.4f}", indent)
    # [D66:END]
```

### Model Forward Pass

```python
def forward(self, x):
    # [D66:START]
    d66_log(f"─── FORWARD {self.__class__.__name__} ───")
    d66_tensor(x, "input")
    # [D66:END]
    
    x = self.layer1(x)
    # [D66:START]
    d66_tensor(x, "after layer1")
    # [D66:END]
    
    x = self.layer2(x)
    # [D66:START]
    d66_tensor(x, "after layer2 (output)")
    # [D66:END]
    
    return x
```

### Training Loop Instrumentation

```python
for epoch in range(epochs):
    # [D66:START]
    d66_log(f"─── EPOCH {epoch+1}/{epochs} ───")
    _d66_epoch_start = time.perf_counter()
    # [D66:END]
    
    for batch_idx, (data, target) in enumerate(dataloader):
        # [D66:START]
        if batch_idx % 100 == 0:
            d66_log(f"  BATCH: [{batch_idx}/{len(dataloader)}]")
            d66_tensor(data, "batch_data")
        # [D66:END]
        
        optimizer.zero_grad()
        output = model(data)
        loss = criterion(output, target)
        
        # [D66:START]
        if batch_idx % 100 == 0:
            d66_log(f"  LOSS: {loss.item():.4f}")
        # [D66:END]
        
        loss.backward()
        
        # [D66:START]
        if batch_idx % 100 == 0:
            # Log gradient norms
            total_norm = sum(p.grad.norm().item()**2 for p in model.parameters() if p.grad is not None)**0.5
            d66_log(f"  GRAD_NORM: {total_norm:.4f}")
        # [D66:END]
        
        optimizer.step()
    
    # [D66:START]
    d66_log(f"─── EPOCH {epoch+1} complete ({time.perf_counter() - _d66_epoch_start:.1f}s) ───")
    # [D66:END]
```

### GPU Memory Tracking

```python
# [D66:START]
import torch
if torch.cuda.is_available():
    d66_log(f"  GPU: {torch.cuda.get_device_name()}")
    d66_log(f"  GPU MEM: allocated={torch.cuda.memory_allocated()/1e9:.2f}GB, "
            f"cached={torch.cuda.memory_reserved()/1e9:.2f}GB")
# [D66:END]
```

---

## TensorFlow/Keras Extensions

### Tensor State Inspection

```python
def d66_tf_tensor(t, name: str, indent: int = 0):
    """Log TensorFlow tensor state."""
    # [D66:START]
    d66_log(f"TENSOR: {name}", indent)
    d66_log(f"  shape: {t.shape.as_list()}", indent)
    d66_log(f"  dtype: {t.dtype.name}", indent)
    if hasattr(t, 'device'):
        d66_log(f"  device: {t.device}", indent)
    # Eager execution values
    if tf.executing_eagerly() and t.shape.num_elements() and t.shape.num_elements() <= 1000:
        d66_log(f"  range: [{tf.reduce_min(t).numpy():.4f}, {tf.reduce_max(t).numpy():.4f}]", indent)
    # [D66:END]
```

### Custom Layer with Debugging

```python
class DebuggedDense(tf.keras.layers.Layer):
    def call(self, inputs):
        # [D66:START]
        d66_log(f"─── LAYER {self.name} ───")
        d66_tf_tensor(inputs, "inputs")
        # [D66:END]
        
        outputs = tf.matmul(inputs, self.kernel) + self.bias
        
        # [D66:START]
        d66_tf_tensor(outputs, "outputs")
        # [D66:END]
        return outputs
```

### Training Callback

```python
class D66Callback(tf.keras.callbacks.Callback):
    def on_epoch_begin(self, epoch, logs=None):
        # [D66:START]
        d66_log(f"─── EPOCH {epoch+1} BEGIN ───")
        # [D66:END]
    
    def on_epoch_end(self, epoch, logs=None):
        # [D66:START]
        d66_log(f"─── EPOCH {epoch+1} END ───")
        for k, v in (logs or {}).items():
            d66_log(f"  {k}: {v:.4f}")
        # [D66:END]
    
    def on_batch_end(self, batch, logs=None):
        # [D66:START]
        if batch % 100 == 0:
            d66_log(f"  BATCH: {batch} loss={logs.get('loss', 'N/A'):.4f}")
        # [D66:END]
```

---

## Google Cloud Storage Extensions

### Client/Authentication State

```python
def d66_gcs_client(client, indent: int = 0):
    """Log GCS client state."""
    # [D66:START]
    d66_log(f"GCS CLIENT:", indent)
    d66_log(f"  project: {client.project}", indent)
    try:
        # Test authentication
        list(client.list_buckets(max_results=1))
        d66_log(f"  auth: OK", indent)
    except Exception as e:
        d66_log(f"  auth: FAILED - {e}", indent)
    # [D66:END]
```

### Blob Operations

```python
def d66_gcs_blob(blob, indent: int = 0):
    """Log GCS blob state."""
    # [D66:START]
    d66_log(f"GCS BLOB: {blob.name}", indent)
    d66_log(f"  bucket: {blob.bucket.name}", indent)
    d66_log(f"  exists: {blob.exists()}", indent)
    if blob.exists():
        blob.reload()
        d66_log(f"  size: {blob.size / 1024:.1f} KB", indent)
        d66_log(f"  content_type: {blob.content_type}", indent)
        d66_log(f"  updated: {blob.updated}", indent)
    # [D66:END]
```

### Upload/Download Tracking

```python
# Upload with progress
# [D66:START]
d66_log(f"─── GCS UPLOAD ───")
d66_log(f"  source: {local_path} ({os.path.getsize(local_path) / 1024:.1f} KB)")
d66_log(f"  dest: gs://{bucket_name}/{blob_name}")
_d66_start = time.perf_counter()
# [D66:END]

blob.upload_from_filename(local_path)

# [D66:START]
d66_log(f"─── GCS UPLOAD complete ({time.perf_counter() - _d66_start:.2f}s) ───")
# [D66:END]

# Download with progress
# [D66:START]
d66_log(f"─── GCS DOWNLOAD ───")
d66_log(f"  source: gs://{bucket_name}/{blob_name}")
d66_log(f"  dest: {local_path}")
_d66_start = time.perf_counter()
# [D66:END]

blob.download_to_filename(local_path)

# [D66:START]
d66_log(f"─── GCS DOWNLOAD complete ({time.perf_counter() - _d66_start:.2f}s, {os.path.getsize(local_path) / 1024:.1f} KB) ───")
# [D66:END]
```

### Batch Operations

```python
# [D66:START]
d66_log(f"─── GCS BATCH LIST ───")
d66_log(f"  bucket: {bucket_name}")
d66_log(f"  prefix: {prefix}")
# [D66:END]

blobs = list(bucket.list_blobs(prefix=prefix))

# [D66:START]
d66_log(f"─── GCS BATCH LIST complete: {len(blobs)} blobs ───")
if blobs:
    total_size = sum(b.size or 0 for b in blobs)
    d66_log(f"  total_size: {total_size / 1024 / 1024:.1f} MB")
    d66_log(f"  first: {blobs[0].name}")
    d66_log(f"  last: {blobs[-1].name}")
# [D66:END]
```

---

## Cleanup

Remove all Debug 66 instrumentation:

```bash
# Remove D66 lines
grep -v "D66" file.py > file_clean.py

# Or use sed
sed -i '/D66/d' file.py

# Remove D66 block comments
sed -i '/# \[D66:START\]/,/# \[D66:END\]/d' file.py
```

## Quick Copy-Paste Templates

### Minimal Function Wrapper
```python
# [D66:START]
d66_log(f"─── ENTER %FNAME% ───"); _d66_t = time.perf_counter()
# [D66:END]
# ... body ...
# [D66:START]
d66_log(f"─── EXIT %FNAME% ({time.perf_counter() - _d66_t:.3f}s) ───")
# [D66:END]
```

### Data Checkpoint
```python
# [D66:START]
d66_log(f"  DATA: %VAR% | {type(%VAR%).__name__} | {_summarize(%VAR%)}")
# [D66:END]
```
