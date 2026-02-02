---
paths:
  - "**/models/**/*.py"
  - "**/training/**/*.py"
  - "**/inference/**/*.py"
  - "**/lightning_modules/**/*.py"
  - "**/callbacks/**/*.py"
  - "**/losses/**/*.py"
---

# Python ML/NN Conventions

Guidelines for PyTorch model development, training, and deployment.

---

## Core Principles

1. **Explicit tensor shapes** - Document shapes in comments: `# [batch, channels, seq_len]`
2. **Type hints everywhere** - All function signatures must be typed
3. **Prefer composition** - Build complex models from simple, tested components
4. **Fail fast** - Validate shapes and assumptions early in forward pass

---

## 1. PyTorch Module Patterns

### 1.1 Standard Module Structure

```python
from dataclasses import dataclass
from typing import Self
import torch
import torch.nn as nn
import torch.nn.functional as F


@dataclass
class EncoderConfig:
    """Configuration for encoder module."""
    input_channels: int = 1
    base_channels: int = 32
    num_levels: int = 4
    kernel_size: int = 7
    dropout: float = 0.1


class Encoder(nn.Module):
    """
    Multi-scale CNN encoder.

    Input: [batch, input_channels, seq_len]
    Output: List of [batch, channels_i, seq_len_i] at each level
    """

    def __init__(self, config: EncoderConfig) -> None:
        super().__init__()
        self.config = config

        # Build layers
        self.levels = nn.ModuleList()
        in_ch = config.input_channels

        for i in range(config.num_levels):
            out_ch = config.base_channels * (2 ** i)
            self.levels.append(
                ConvBlock(in_ch, out_ch, config.kernel_size, stride=2)
            )
            in_ch = out_ch

    def forward(self, x: torch.Tensor) -> list[torch.Tensor]:
        """
        Args:
            x: Input tensor [batch, input_channels, seq_len]

        Returns:
            List of feature maps at each resolution level
        """
        features = []
        for level in self.levels:
            x = level(x)
            features.append(x)
        return features
```

### 1.2 Normalization Choice

| Scenario | Recommended | Why |
|----------|-------------|-----|
| Variable batch size | GroupNorm | Batch-independent statistics |
| Fixed large batch | BatchNorm | Fastest, well-understood |
| Sequence data | LayerNorm | Per-sample normalization |
| Very deep networks | LayerNorm | More stable gradients |

```python
def get_norm_layer(
    num_channels: int,
    norm_type: str = "group",
    num_groups: int = 8,
) -> nn.Module:
    """Get appropriate normalization layer."""
    if norm_type == "group":
        return nn.GroupNorm(num_groups, num_channels)
    elif norm_type == "batch":
        return nn.BatchNorm1d(num_channels)
    elif norm_type == "layer":
        return nn.LayerNorm(num_channels)
    elif norm_type == "instance":
        return nn.InstanceNorm1d(num_channels)
    else:
        raise ValueError(f"Unknown norm type: {norm_type}")
```

### 1.3 Activation Functions

**Default: GELU** (smoother gradients than ReLU)

```python
class ConvBlock(nn.Module):
    """Standard convolution block with norm and activation."""

    def __init__(
        self,
        in_channels: int,
        out_channels: int,
        kernel_size: int = 7,
        stride: int = 1,
        groups: int = 8,
    ) -> None:
        super().__init__()
        padding = kernel_size // 2

        self.conv = nn.Conv1d(
            in_channels, out_channels, kernel_size,
            stride=stride, padding=padding
        )
        self.norm = nn.GroupNorm(groups, out_channels)
        self.act = nn.GELU()

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.act(self.norm(self.conv(x)))
```

---

## 2. Attention Mechanism Patterns

### Decision Matrix

| Mechanism | Complexity | Use When |
|-----------|------------|----------|
| Full self-attention | O(n²) | n < 1000, need global context |
| Local window | O(n × w) | Spatial locality matters (isotopes) |
| Linear (Linformer) | O(n × k) | Long sequences, learned compression |
| Performer | O(n × r) | Long sequences, position-independent |
| Cross-scale | Varies | Multi-resolution features |

### 2.1 Full Self-Attention

**Only use when sequence length < 1000**

```python
class SelfAttention(nn.Module):
    """Standard multi-head self-attention. O(n²) complexity."""

    def __init__(
        self,
        embed_dim: int,
        num_heads: int = 8,
        dropout: float = 0.1,
    ) -> None:
        super().__init__()
        self.attention = nn.MultiheadAttention(
            embed_dim, num_heads,
            dropout=dropout,
            batch_first=True
        )
        self.norm = nn.LayerNorm(embed_dim)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Args:
            x: [batch, seq_len, embed_dim]
        """
        attn_out, _ = self.attention(x, x, x)
        return self.norm(x + attn_out)
```

### 2.2 Local Window Attention

**PREFERRED for spatial patterns like isotopes**

```python
class LocalWindowAttention(nn.Module):
    """
    Self-attention with local window constraint.

    O(n × window_size) complexity.
    Useful when patterns are spatially localized.
    """

    def __init__(
        self,
        embed_dim: int,
        num_heads: int = 8,
        window_size: int = 64,
        dropout: float = 0.1,
    ) -> None:
        super().__init__()
        self.window_size = window_size
        self.attention = nn.MultiheadAttention(
            embed_dim, num_heads,
            dropout=dropout,
            batch_first=True
        )
        self.norm = nn.LayerNorm(embed_dim)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Args:
            x: [batch, seq_len, embed_dim]
        """
        batch, seq_len, embed_dim = x.shape

        # Pad to multiple of window_size
        pad_len = (self.window_size - seq_len % self.window_size) % self.window_size
        if pad_len > 0:
            x = F.pad(x, (0, 0, 0, pad_len))

        # Reshape into windows: [batch * num_windows, window_size, embed_dim]
        padded_len = x.size(1)
        num_windows = padded_len // self.window_size
        x_windows = x.view(batch, num_windows, self.window_size, embed_dim)
        x_windows = x_windows.view(batch * num_windows, self.window_size, embed_dim)

        # Apply attention within each window
        attn_out, _ = self.attention(x_windows, x_windows, x_windows)

        # Reshape back
        attn_out = attn_out.view(batch, num_windows, self.window_size, embed_dim)
        attn_out = attn_out.view(batch, padded_len, embed_dim)

        # Remove padding and apply residual
        attn_out = attn_out[:, :seq_len]
        x = x[:, :seq_len]

        return self.norm(x + attn_out)
```

### 2.3 Linear Attention (Linformer)

**Use for long sequences with learned compression**

```python
class LinearAttention(nn.Module):
    """
    Linformer-style linear attention.

    Projects keys and values to lower dimension k << n.
    O(n × k) complexity instead of O(n²).
    """

    def __init__(
        self,
        embed_dim: int,
        num_heads: int = 8,
        seq_len: int = 2048,
        k: int = 256,
        dropout: float = 0.1,
    ) -> None:
        super().__init__()
        self.embed_dim = embed_dim
        self.num_heads = num_heads
        self.head_dim = embed_dim // num_heads
        self.k = k

        # Standard QKV projections
        self.q_proj = nn.Linear(embed_dim, embed_dim)
        self.k_proj = nn.Linear(embed_dim, embed_dim)
        self.v_proj = nn.Linear(embed_dim, embed_dim)
        self.out_proj = nn.Linear(embed_dim, embed_dim)

        # Key/value dimension reduction (Linformer core)
        self.k_reduce = nn.Linear(seq_len, k)
        self.v_reduce = nn.Linear(seq_len, k)

        self.dropout = nn.Dropout(dropout)
        self.norm = nn.LayerNorm(embed_dim)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Args:
            x: [batch, seq_len, embed_dim]
        """
        batch_size, seq_len, _ = x.shape

        # Project to Q, K, V
        Q = self.q_proj(x)  # [batch, seq_len, embed_dim]
        K = self.k_proj(x)
        V = self.v_proj(x)

        # Reshape for multi-head: [batch, num_heads, seq_len, head_dim]
        Q = Q.view(batch_size, seq_len, self.num_heads, self.head_dim).transpose(1, 2)
        K = K.view(batch_size, seq_len, self.num_heads, self.head_dim).transpose(1, 2)
        V = V.view(batch_size, seq_len, self.num_heads, self.head_dim).transpose(1, 2)

        # Reduce K and V: [batch, heads, head_dim, seq] -> [batch, heads, head_dim, k]
        K = self.k_reduce(K.transpose(-2, -1)).transpose(-2, -1)  # [batch, heads, k, head_dim]
        V = self.v_reduce(V.transpose(-2, -1)).transpose(-2, -1)  # [batch, heads, k, head_dim]

        # Attention: [batch, heads, seq, head_dim] @ [batch, heads, head_dim, k]
        attn_weights = torch.matmul(Q, K.transpose(-2, -1)) / (self.head_dim ** 0.5)
        attn_weights = F.softmax(attn_weights, dim=-1)
        attn_weights = self.dropout(attn_weights)

        # [batch, heads, seq, k] @ [batch, heads, k, head_dim]
        output = torch.matmul(attn_weights, V)

        # Reshape back: [batch, seq_len, embed_dim]
        output = output.transpose(1, 2).contiguous().view(batch_size, seq_len, self.embed_dim)
        output = self.out_proj(output)

        return self.norm(x + output)
```

### 2.4 Cross-Scale Attention

```python
class CrossScaleAttention(nn.Module):
    """
    Attention between features at different scales.

    Small scale queries, medium/large scales provide context.
    """

    def __init__(
        self,
        embed_dim: int,
        num_heads: int = 8,
    ) -> None:
        super().__init__()
        self.cross_attn = nn.MultiheadAttention(
            embed_dim, num_heads, batch_first=True
        )
        self.norm = nn.LayerNorm(embed_dim)

    def forward(
        self,
        query: torch.Tensor,
        context: torch.Tensor,
    ) -> torch.Tensor:
        """
        Args:
            query: [batch, seq_q, embed_dim] - scale to update
            context: [batch, seq_c, embed_dim] - context from other scales
        """
        attn_out, _ = self.cross_attn(query, context, context)
        return self.norm(query + attn_out)
```

---

## 3. Loss Function Patterns

### 3.1 Focal Loss (Class Imbalance)

**REQUIRED for imbalanced classification**

```python
class FocalLoss(nn.Module):
    """
    Focal loss for addressing class imbalance.

    FL(p_t) = -alpha_t * (1 - p_t)^gamma * log(p_t)

    Args:
        alpha: Weighting factor for positive class (0.25 typical)
        gamma: Focusing parameter (2.0 typical)
    """

    def __init__(
        self,
        alpha: float = 0.25,
        gamma: float = 2.0,
        reduction: str = "mean",
    ) -> None:
        super().__init__()
        self.alpha = alpha
        self.gamma = gamma
        self.reduction = reduction

    def forward(
        self,
        pred: torch.Tensor,
        target: torch.Tensor,
    ) -> torch.Tensor:
        """
        Args:
            pred: Predictions in [0, 1], shape [batch, ...]
            target: Binary targets, shape [batch, ...]
        """
        # Clamp for numerical stability
        pred = pred.clamp(1e-7, 1 - 1e-7)

        # Binary cross entropy (unreduced)
        bce = F.binary_cross_entropy(pred, target, reduction='none')

        # Focal weight
        pt = torch.where(target == 1, pred, 1 - pred)
        focal_weight = (1 - pt) ** self.gamma

        # Alpha weight
        alpha_weight = torch.where(target == 1, self.alpha, 1 - self.alpha)

        loss = alpha_weight * focal_weight * bce

        if self.reduction == "mean":
            return loss.mean()
        elif self.reduction == "sum":
            return loss.sum()
        return loss
```

### 3.2 Permutation-Invariant Loss (Set Prediction)

**REQUIRED for predicting sets (e.g., multiple series)**

```python
from scipy.optimize import linear_sum_assignment

class HungarianLoss(nn.Module):
    """
    Permutation-invariant loss using Hungarian matching.

    For set prediction where order doesn't matter.
    Matches predicted items to targets optimally before computing loss.
    """

    def __init__(
        self,
        base_loss: nn.Module | None = None,
    ) -> None:
        super().__init__()
        self.base_loss = base_loss or nn.MSELoss(reduction='none')

    def forward(
        self,
        pred: torch.Tensor,
        target: torch.Tensor,
    ) -> torch.Tensor:
        """
        Args:
            pred: [batch, n_items, item_dim]
            target: [batch, n_items, item_dim]
        """
        batch_size = pred.size(0)
        total_loss = 0.0

        for b in range(batch_size):
            # Compute pairwise cost matrix
            # cost[i, j] = loss(pred[i], target[j])
            cost_matrix = torch.zeros(pred.size(1), target.size(1))

            for i in range(pred.size(1)):
                for j in range(target.size(1)):
                    cost_matrix[i, j] = self.base_loss(
                        pred[b, i], target[b, j]
                    ).sum()

            # Hungarian algorithm for optimal matching
            row_ind, col_ind = linear_sum_assignment(cost_matrix.detach().cpu().numpy())

            # Compute loss on matched pairs
            matched_pred = pred[b, row_ind]
            matched_target = target[b, col_ind]
            total_loss += self.base_loss(matched_pred, matched_target).mean()

        return total_loss / batch_size


# Faster differentiable approximation using Sinkhorn
class SinkhornLoss(nn.Module):
    """
    Differentiable approximation to Hungarian matching.

    Uses Sinkhorn iterations for soft assignment.
    Fully differentiable unlike scipy-based Hungarian.
    """

    def __init__(
        self,
        n_iters: int = 100,
        temperature: float = 0.1,
    ) -> None:
        super().__init__()
        self.n_iters = n_iters
        self.temperature = temperature

    def forward(
        self,
        pred: torch.Tensor,
        target: torch.Tensor,
    ) -> torch.Tensor:
        # Compute cost matrix
        cost = torch.cdist(pred, target, p=2)  # [batch, n_pred, n_target]

        # Sinkhorn iterations
        log_alpha = -cost / self.temperature
        for _ in range(self.n_iters):
            log_alpha = log_alpha - torch.logsumexp(log_alpha, dim=-1, keepdim=True)
            log_alpha = log_alpha - torch.logsumexp(log_alpha, dim=-2, keepdim=True)

        # Soft assignment
        assignment = log_alpha.exp()

        # Weighted loss
        loss = (assignment * cost).sum(dim=(-1, -2)).mean()

        return loss
```

### 3.3 Ordinal Regression Loss

**Use for ordered categories (1, 2, 3, 4, 5 series)**

```python
class OrdinalRegressionLoss(nn.Module):
    """
    Ordinal regression using cumulative logits.

    For ordered categories where 3 is "closer" to 2 than to 5.
    Predicts P(Y >= k) for each threshold k.
    """

    def __init__(self, num_classes: int) -> None:
        super().__init__()
        self.num_classes = num_classes

    def forward(
        self,
        logits: torch.Tensor,
        target: torch.Tensor,
    ) -> torch.Tensor:
        """
        Args:
            logits: [batch, num_classes - 1] cumulative logits
            target: [batch] integer class labels 0 to num_classes-1
        """
        # Convert target to cumulative binary: for class k, Y >= 0, Y >= 1, ..., Y >= k are 1
        # Shape: [batch, num_classes - 1]
        cumulative_target = (
            target.unsqueeze(1) >= torch.arange(1, self.num_classes, device=target.device)
        ).float()

        # Binary cross-entropy on each threshold
        loss = F.binary_cross_entropy_with_logits(
            logits, cumulative_target, reduction='mean'
        )

        return loss

    @staticmethod
    def predict(logits: torch.Tensor) -> torch.Tensor:
        """Convert cumulative logits to class prediction."""
        cumulative_probs = torch.sigmoid(logits)

        # P(Y = k) = P(Y >= k) - P(Y >= k+1)
        # Predicted class = number of thresholds exceeded
        predicted = (cumulative_probs > 0.5).sum(dim=-1)

        return predicted
```

### 3.4 Existence Masking (CORRECTED)

**CRITICAL: The cumsum approach in some codebases is INVERTED**

```python
def compute_existence_mask(
    num_items_probs: torch.Tensor,
    max_items: int,
) -> torch.Tensor:
    """
    Compute soft mask for item existence.

    P(item_i exists) = P(num_items >= i)
                     = 1 - P(num_items < i)
                     = 1 - sum(P(num_items = k) for k < i)

    Args:
        num_items_probs: [batch, max_items] probability of exactly k items
        max_items: Maximum number of items

    Returns:
        existence_mask: [batch, max_items] probability each item exists
    """
    # CORRECT implementation:
    # P(exists_0) = 1 (at least one item always)
    # P(exists_1) = P(n >= 2) = 1 - P(n = 1)
    # P(exists_2) = P(n >= 3) = 1 - P(n = 1) - P(n = 2)

    cumsum = torch.cumsum(num_items_probs, dim=1)

    # P(item_i exists) = 1 - cumsum of probabilities for fewer items
    existence_mask = torch.ones_like(num_items_probs)
    existence_mask[:, 1:] = 1 - cumsum[:, :-1]

    return existence_mask


# WRONG (commonly seen bug):
def compute_existence_mask_WRONG(num_items_probs):
    # This is INVERTED - suppresses early items, keeps late ones
    cumsum = torch.cumsum(num_items_probs, dim=1)
    return cumsum  # WRONG!
```

---

## 4. Output Head Patterns

### 4.1 Multi-Label Classification

**Use sigmoid, not softmax**

```python
class MultiLabelHead(nn.Module):
    """
    Multi-label classification head.

    Each class is independent (not mutually exclusive).
    Example: multiple charge states present simultaneously.
    """

    def __init__(self, in_features: int, num_labels: int) -> None:
        super().__init__()
        self.classifier = nn.Linear(in_features, num_labels)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """Returns probabilities for each label independently."""
        logits = self.classifier(x)
        return torch.sigmoid(logits)  # NOT softmax!

    def predict(self, x: torch.Tensor, threshold: float = 0.5) -> torch.Tensor:
        """Binary predictions for each label."""
        probs = self.forward(x)
        return (probs > threshold).long()
```

### 4.2 Uncertainty Estimation

```python
class UncertaintyHead(nn.Module):
    """
    Output with uncertainty estimation.

    Predicts mean and log-variance for Gaussian output.
    """

    def __init__(self, in_features: int) -> None:
        super().__init__()
        self.mean_head = nn.Linear(in_features, 1)
        self.log_var_head = nn.Linear(in_features, 1)

    def forward(
        self,
        x: torch.Tensor,
    ) -> tuple[torch.Tensor, torch.Tensor]:
        """
        Returns:
            mean: Predicted value
            std: Uncertainty (standard deviation)
        """
        mean = self.mean_head(x)
        log_var = self.log_var_head(x)

        # Clamp log_var for numerical stability
        log_var = torch.clamp(log_var, min=-10, max=10)
        std = torch.exp(0.5 * log_var)

        return mean, std


class GaussianNLLLoss(nn.Module):
    """Negative log-likelihood for Gaussian with predicted variance."""

    def forward(
        self,
        mean: torch.Tensor,
        std: torch.Tensor,
        target: torch.Tensor,
    ) -> torch.Tensor:
        # NLL = 0.5 * (log(var) + (target - mean)^2 / var)
        var = std ** 2
        loss = 0.5 * (torch.log(var) + (target - mean) ** 2 / var)
        return loss.mean()
```

---

## 5. Lightning Module Patterns

### 5.1 Standard Structure

```python
import pytorch_lightning as pl
from torch.optim import AdamW
from torch.optim.lr_scheduler import CosineAnnealingLR


class BaseModule(pl.LightningModule):
    """Base Lightning module with standard patterns."""

    def __init__(
        self,
        model: nn.Module,
        learning_rate: float = 1e-4,
        weight_decay: float = 1e-5,
        warmup_epochs: int = 5,
        max_epochs: int = 100,
    ) -> None:
        super().__init__()
        self.save_hyperparameters(ignore=['model'])

        self.model = model
        self.learning_rate = learning_rate
        self.weight_decay = weight_decay
        self.warmup_epochs = warmup_epochs
        self.max_epochs = max_epochs

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.model(x)

    def training_step(
        self,
        batch: tuple[torch.Tensor, ...],
        batch_idx: int,
    ) -> torch.Tensor:
        x, y = batch
        y_hat = self(x)
        loss = self.compute_loss(y_hat, y)

        self.log('train/loss', loss, prog_bar=True)
        return loss

    def validation_step(
        self,
        batch: tuple[torch.Tensor, ...],
        batch_idx: int,
    ) -> None:
        x, y = batch
        y_hat = self(x)
        loss = self.compute_loss(y_hat, y)
        metrics = self.compute_metrics(y_hat, y)

        self.log('val/loss', loss, prog_bar=True)
        for name, value in metrics.items():
            self.log(f'val/{name}', value)

    def configure_optimizers(self):
        optimizer = AdamW(
            self.parameters(),
            lr=self.learning_rate,
            weight_decay=self.weight_decay,
        )

        scheduler = CosineAnnealingLR(
            optimizer,
            T_max=self.max_epochs - self.warmup_epochs,
        )

        return {
            'optimizer': optimizer,
            'lr_scheduler': {
                'scheduler': scheduler,
                'interval': 'epoch',
            },
        }

    def compute_loss(self, y_hat, y) -> torch.Tensor:
        """Override in subclass."""
        raise NotImplementedError

    def compute_metrics(self, y_hat, y) -> dict[str, float]:
        """Override in subclass."""
        return {}
```

### 5.2 Multi-Task Module

```python
class MultiTaskModule(pl.LightningModule):
    """Module with multiple task heads and weighted loss."""

    def __init__(
        self,
        model: nn.Module,
        task_weights: dict[str, float],
        use_gradnorm: bool = False,
    ) -> None:
        super().__init__()
        self.model = model
        self.task_weights = task_weights
        self.use_gradnorm = use_gradnorm

        if use_gradnorm:
            # Learnable loss weights
            self.log_weights = nn.Parameter(
                torch.zeros(len(task_weights))
            )

    def compute_weighted_loss(
        self,
        losses: dict[str, torch.Tensor],
    ) -> torch.Tensor:
        """Compute weighted sum of task losses."""
        if self.use_gradnorm:
            weights = F.softmax(self.log_weights, dim=0)
            weight_dict = dict(zip(self.task_weights.keys(), weights))
        else:
            weight_dict = self.task_weights

        total = sum(
            weight_dict[name] * loss
            for name, loss in losses.items()
        )

        # Log individual losses
        for name, loss in losses.items():
            self.log(f'train/{name}_loss', loss)

        return total
```

---

## 6. Contrastive Learning Patterns

### 6.1 SimCLR-Style

```python
class ContrastiveModule(nn.Module):
    """
    SimCLR-style contrastive learning.

    Learns representations where similar samples cluster.
    """

    def __init__(
        self,
        encoder: nn.Module,
        projection_dim: int = 256,
        temperature: float = 0.07,
    ) -> None:
        super().__init__()
        self.encoder = encoder
        self.temperature = temperature

        # Projection head (discarded after pretraining)
        encoder_dim = self._get_encoder_dim()
        self.projector = nn.Sequential(
            nn.Linear(encoder_dim, encoder_dim),
            nn.GELU(),
            nn.Linear(encoder_dim, projection_dim),
        )

    def forward(
        self,
        x1: torch.Tensor,
        x2: torch.Tensor,
    ) -> torch.Tensor:
        """
        Compute NT-Xent loss between two augmented views.

        Args:
            x1, x2: Two augmented views of same batch
        """
        # Encode and project
        z1 = F.normalize(self.projector(self.encoder(x1)), dim=1)
        z2 = F.normalize(self.projector(self.encoder(x2)), dim=1)

        return self.nt_xent_loss(z1, z2)

    def nt_xent_loss(
        self,
        z1: torch.Tensor,
        z2: torch.Tensor,
    ) -> torch.Tensor:
        """Normalized Temperature-scaled Cross Entropy Loss."""
        batch_size = z1.size(0)
        device = z1.device

        # Concatenate: [2*batch, dim]
        z = torch.cat([z1, z2], dim=0)

        # Similarity matrix: [2*batch, 2*batch]
        sim = torch.mm(z, z.t()) / self.temperature

        # Mask self-similarity
        mask = torch.eye(2 * batch_size, device=device).bool()
        sim.masked_fill_(mask, float('-inf'))

        # Positive pairs: (i, i+batch) and (i+batch, i)
        labels = torch.cat([
            torch.arange(batch_size, 2 * batch_size, device=device),
            torch.arange(batch_size, device=device),
        ])

        return F.cross_entropy(sim, labels)
```

---

## 7. ONNX Export Patterns

```python
import torch.onnx

def export_to_onnx(
    model: nn.Module,
    output_path: str,
    input_shape: tuple[int, ...] = (1, 1, 2048),
    dynamic_axes: dict | None = None,
    opset_version: int = 17,
) -> None:
    """
    Export PyTorch model to ONNX.

    Args:
        model: Trained model in eval mode
        output_path: Path for .onnx file
        input_shape: Example input shape
        dynamic_axes: Axes that can vary (batch, seq_len)
        opset_version: ONNX opset version
    """
    model.eval()

    if dynamic_axes is None:
        dynamic_axes = {
            'input': {0: 'batch', 2: 'seq_len'},
            'output': {0: 'batch'},
        }

    dummy_input = torch.randn(*input_shape)

    torch.onnx.export(
        model,
        dummy_input,
        output_path,
        input_names=['input'],
        output_names=['output'],
        dynamic_axes=dynamic_axes,
        opset_version=opset_version,
        do_constant_folding=True,
    )

    # Validate
    import onnx
    onnx_model = onnx.load(output_path)
    onnx.checker.check_model(onnx_model)


def validate_onnx_output(
    pytorch_model: nn.Module,
    onnx_path: str,
    test_input: torch.Tensor,
    rtol: float = 1e-3,
    atol: float = 1e-5,
) -> bool:
    """Validate ONNX output matches PyTorch."""
    import numpy as np
    import onnxruntime as ort

    # PyTorch inference
    pytorch_model.eval()
    with torch.no_grad():
        pytorch_output = pytorch_model(test_input).numpy()

    # ONNX inference
    session = ort.InferenceSession(onnx_path)
    onnx_output = session.run(
        None,
        {'input': test_input.numpy()}
    )[0]

    # Compare
    return np.allclose(pytorch_output, onnx_output, rtol=rtol, atol=atol)
```

---

## 8. Calibration Patterns

### 8.1 Temperature Scaling

```python
class TemperatureScaling(nn.Module):
    """
    Post-hoc temperature scaling for calibration.

    Learns a single temperature parameter on validation set.
    """

    def __init__(self) -> None:
        super().__init__()
        self.temperature = nn.Parameter(torch.ones(1))

    def forward(self, logits: torch.Tensor) -> torch.Tensor:
        """Scale logits by learned temperature."""
        return logits / self.temperature

    def calibrate(
        self,
        logits: torch.Tensor,
        labels: torch.Tensor,
        max_iter: int = 100,
    ) -> None:
        """Fit temperature on validation data."""
        optimizer = torch.optim.LBFGS([self.temperature], max_iter=max_iter)

        def closure():
            optimizer.zero_grad()
            scaled = self.forward(logits)
            loss = F.cross_entropy(scaled, labels)
            loss.backward()
            return loss

        optimizer.step(closure)


def expected_calibration_error(
    probs: torch.Tensor,
    labels: torch.Tensor,
    n_bins: int = 15,
) -> float:
    """
    Compute Expected Calibration Error.

    ECE = sum_b (|B_b| / n) * |accuracy(B_b) - confidence(B_b)|

    Lower is better. < 0.05 is well-calibrated.
    """
    confidences, predictions = probs.max(dim=1)
    accuracies = predictions.eq(labels)

    ece = 0.0
    for bin_lower in np.linspace(0, 1 - 1/n_bins, n_bins):
        bin_upper = bin_lower + 1/n_bins
        in_bin = (confidences > bin_lower) & (confidences <= bin_upper)
        prop_in_bin = in_bin.float().mean()

        if prop_in_bin > 0:
            avg_confidence = confidences[in_bin].mean()
            avg_accuracy = accuracies[in_bin].float().mean()
            ece += prop_in_bin * abs(avg_accuracy - avg_confidence)

    return ece.item()
```

---

## 9. Anti-Patterns to Avoid

### DO NOT:

1. **Use `Any` type for tensor shapes** - Be explicit: `torch.Tensor  # [B, C, L]`
2. **Use BatchNorm with small/variable batch** - Use GroupNorm or LayerNorm
3. **Use full attention when seq_len > 2000** - Use local or linear attention
4. **Use MSE for imbalanced classification** - Use focal loss
5. **Ignore permutation invariance in set prediction** - Use Hungarian matching
6. **Use cumsum directly for existence masking** - It's inverted!
7. **Skip ONNX validation** - Always verify output matches

### Common Mistakes:

```python
# BAD: Attention on long sequence
attn = nn.MultiheadAttention(dim, heads)
out = attn(x, x, x)  # O(n²) for n=2500 → 6.25M operations

# GOOD: Local attention
attn = LocalWindowAttention(dim, heads, window_size=64)
out = attn(x)  # O(n × 64) → 160K operations

# BAD: Softmax for multi-label
probs = F.softmax(logits, dim=-1)  # Forces sum=1

# GOOD: Sigmoid for multi-label
probs = torch.sigmoid(logits)  # Each independent

# BAD: Wrong existence mask
mask = torch.cumsum(num_probs, dim=1)  # INVERTED

# GOOD: Correct existence mask
mask = torch.ones_like(num_probs)
mask[:, 1:] = 1 - torch.cumsum(num_probs, dim=1)[:, :-1]
```

---

## References

- Phase 1 Critical Analysis: Sections 3-9
- overall-plan.md: Sections 5-6
- PyTorch Lightning documentation
- ONNX export best practices
