---
paths:
  - "**/data/**/*.py"
  - "**/preprocessing/**/*.py"
  - "**/transforms/**/*.py"
  - "**/augmentation/**/*.py"
  - "**/loaders/**/*.py"
  - "**/serialization/**/*.py"
---

# Python Data Science Conventions

Guidelines for signal processing, spectral data handling, and statistical methods in Python.

---

## Core Principles

1. **Understand your data's statistical properties** before choosing transforms
2. **Preserve information** unless explicitly trading accuracy for speed
3. **Make decisions explicit** - document why you chose a particular approach
4. **Validate assumptions** - Poisson assumption, noise model, resolution requirements

---

## 1. Variance Stabilization Transforms (VST)

### Decision Matrix

| Data Type | Noise Model | Recommended VST | When to Use |
|-----------|-------------|-----------------|-------------|
| Count data (low counts) | Poisson | Anscombe | x > 0, most bins < 100 |
| Count data (high counts) | Poisson | Square root | x >> 1, prototyping |
| Mixed noise | Poisson + Gaussian | GAT | Production with calibration |
| Unknown distribution | Unknown | DAIN (learned) | Large unlabeled corpus available |

### 1.1 Square Root Transform

**Use for:** Quick prototyping, high-count regimes

```python
def sqrt_transform(x: np.ndarray) -> np.ndarray:
    """Simple variance stabilization for Poisson data."""
    return np.sqrt(np.maximum(x, 0))

def sqrt_inverse(y: np.ndarray) -> np.ndarray:
    """Inverse of square root transform."""
    return y ** 2
```

### 1.2 Anscombe Transform

**Use for:** Poisson-distributed data, proper VST

```python
def anscombe_transform(x: np.ndarray) -> np.ndarray:
    """
    Anscombe transform for Poisson data.

    Transforms Poisson-distributed data to approximately Gaussian
    with constant variance ~1.

    Valid for x >= 0. Most effective when x > 4.
    """
    return 2.0 * np.sqrt(np.maximum(x, 0) + 3/8)

def anscombe_inverse(y: np.ndarray) -> np.ndarray:
    """
    Inverse Anscombe transform.

    Note: This is the algebraic inverse. For unbiased estimation,
    use the exact unbiased inverse for low counts.
    """
    return np.maximum((y / 2) ** 2 - 3/8, 0)

def anscombe_inverse_exact(y: np.ndarray) -> np.ndarray:
    """
    Exact unbiased inverse Anscombe transform.

    Better for low-count data where bias matters.
    """
    return (1/4) * y**2 + (1/4) * np.sqrt(3/2) * y**(-1) - (11/8) * y**(-2) + \
           (5/8) * np.sqrt(3/2) * y**(-3) - 1/8
```

### 1.3 Generalized Anscombe Transform (GAT)

**Use for:** Mixed Poisson-Gaussian noise (production systems)

```python
def generalized_anscombe_transform(
    x: np.ndarray,
    gain: float = 1.0,
    sigma: float = 0.0,
    offset: float = 0.0,
) -> np.ndarray:
    """
    GAT for mixed Poisson-Gaussian noise.

    Args:
        x: Input signal
        gain: Detector gain (electrons per count)
        sigma: Gaussian noise std (in electrons)
        offset: Detector offset

    The noise model is: y = Poisson(gain * x) + Gaussian(0, sigma) + offset
    """
    b = gain
    inner = b * x + (3/8) * b**2 + sigma**2 - b * offset
    return (2 / b) * np.sqrt(np.maximum(inner, 0))

def estimate_noise_parameters(
    dark_frames: np.ndarray,
    flat_frames: np.ndarray,
) -> tuple[float, float, float]:
    """
    Estimate GAT parameters from calibration data.

    Args:
        dark_frames: Stack of dark frames [n_frames, height, width]
        flat_frames: Stack of flat field frames at various intensities

    Returns:
        gain, sigma, offset
    """
    # Offset from dark frame mean
    offset = np.mean(dark_frames)

    # Sigma from dark frame std
    sigma = np.std(dark_frames)

    # Gain from photon transfer curve (variance vs mean)
    means = np.array([np.mean(f) for f in flat_frames])
    variances = np.array([np.var(f) for f in flat_frames])
    gain = np.polyfit(means - offset, variances - sigma**2, 1)[0]

    return gain, sigma, offset
```

### 1.4 DAIN (Learned Normalization)

**Use for:** When you have large unlabeled corpus and unknown noise model

```python
import torch
import torch.nn as nn

class DAIN(nn.Module):
    """
    Data Adaptive Input Normalization.

    Learns instance-specific normalization from data.
    Ref: Passalis et al., "Deep Adaptive Input Normalization for Time Series"
    """

    def __init__(self, input_dim: int, eps: float = 1e-8):
        super().__init__()
        self.eps = eps

        # Learnable parameters for adaptive normalization
        self.mean_layer = nn.Linear(input_dim, input_dim)
        self.std_layer = nn.Linear(input_dim, input_dim)
        self.gate = nn.Sequential(
            nn.Linear(input_dim, input_dim),
            nn.Sigmoid(),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        # x: [batch, seq_len] or [batch, channels, seq_len]

        # Compute adaptive statistics
        mean = self.mean_layer(x)
        std = torch.abs(self.std_layer(x)) + self.eps

        # Normalize
        normalized = (x - mean) / std

        # Gated combination with original
        gate = self.gate(x)
        return gate * normalized + (1 - gate) * x
```

---

## 2. Binning Strategies

### Decision Matrix

| Method | Complexity | Peak Shape | Speed | Use When |
|--------|------------|------------|-------|----------|
| Histogram | O(n_peaks) | Lost | Fastest | Quick exploration |
| Gaussian (vectorized) | O(n_peaks × n_bins) | Preserved | Medium | Production |
| Linear interpolation | O(n_peaks) | Approximated | Fast | Balance of speed/quality |
| Cubic spline | O(n_peaks) | Best | Medium | Highest accuracy needed |

### 2.1 Histogram Binning

```python
def histogram_bin(
    mz: np.ndarray,
    intensity: np.ndarray,
    bin_edges: np.ndarray,
) -> np.ndarray:
    """Fast histogram binning. Loses sub-bin information."""
    binned, _ = np.histogram(mz, bins=bin_edges, weights=intensity)
    return binned
```

### 2.2 Gaussian-Weighted Binning (Vectorized)

**PREFERRED for production**

```python
def gaussian_bin_vectorized(
    mz: np.ndarray,
    intensity: np.ndarray,
    bin_centers: np.ndarray,
    sigma_da: float = 0.05,
) -> np.ndarray:
    """
    Gaussian-weighted binning preserving peak shape.

    Args:
        mz: m/z values [n_peaks]
        intensity: Intensity values [n_peaks]
        bin_centers: Bin center m/z values [n_bins]
        sigma_da: Gaussian sigma in Daltons

    Returns:
        Binned spectrum [n_bins]

    Memory: O(n_peaks × n_bins) - chunk if memory-limited
    """
    # Compute distances: [n_peaks, n_bins]
    distances = np.abs(mz[:, None] - bin_centers[None, :])

    # Gaussian weights (truncate at 3 sigma for efficiency)
    weights = np.exp(-0.5 * (distances / sigma_da) ** 2)
    weights[distances > 3 * sigma_da] = 0

    # Weighted sum
    binned = (intensity[:, None] * weights).sum(axis=0)

    return binned


def gaussian_bin_chunked(
    mz: np.ndarray,
    intensity: np.ndarray,
    bin_centers: np.ndarray,
    sigma_da: float = 0.05,
    chunk_size: int = 10000,
) -> np.ndarray:
    """Memory-efficient chunked Gaussian binning for large spectra."""
    binned = np.zeros(len(bin_centers))

    for i in range(0, len(mz), chunk_size):
        chunk_mz = mz[i:i + chunk_size]
        chunk_int = intensity[i:i + chunk_size]

        distances = np.abs(chunk_mz[:, None] - bin_centers[None, :])
        weights = np.exp(-0.5 * (distances / sigma_da) ** 2)
        weights[distances > 3 * sigma_da] = 0

        binned += (chunk_int[:, None] * weights).sum(axis=0)

    return binned
```

### 2.3 Adaptive Sigma Selection

```python
def estimate_optimal_sigma(
    mz: np.ndarray,
    intensity: np.ndarray,
    expected_charge_range: tuple[int, int] = (1, 10),
) -> float:
    """
    Estimate optimal Gaussian sigma based on peak density.

    For isotope patterns:
    - z=1: spacing ~1 Da, need sigma < 0.3 Da
    - z=10: spacing ~0.1 Da, need sigma < 0.03 Da

    Returns conservative sigma that works for highest expected charge.
    """
    min_charge, max_charge = expected_charge_range
    min_spacing = 1.0 / max_charge  # Daltons

    # Sigma should be < 1/3 of minimum spacing to avoid overlap
    return min_spacing / 3


def adaptive_sigma_by_region(
    mz_value: float,
    resolution: float = 60000,
) -> float:
    """
    Compute sigma based on instrument resolution.

    FWHM = mz / resolution
    sigma = FWHM / 2.355 (for Gaussian peaks)
    """
    fwhm = mz_value / resolution
    return fwhm / 2.355
```

---

## 3. Baseline Correction

### 3.1 SNIP with Smooth Region Transitions

**PREFERRED approach**

```python
from pybaselines.morphological import snip
from scipy.ndimage import gaussian_filter1d

def adaptive_baseline_correction(
    spectrum: np.ndarray,
    bin_width_da: float = 0.1,
    mz_range: tuple[float, float] = (50, 2500),
) -> np.ndarray:
    """
    m/z-dependent baseline correction with smooth transitions.

    Different regions need different SNIP windows:
    - Low m/z (50-500): Narrow peaks, small window
    - Mid m/z (500-1500): Medium peaks, medium window
    - High m/z (1500-2500): Broad envelopes, large window
    """
    n_bins = len(spectrum)
    bins_per_da = 1 / bin_width_da

    # Define window sizes in bins
    windows = {
        'low': int(30 * bins_per_da / 10),   # ~30 Da effective
        'mid': int(60 * bins_per_da / 10),   # ~60 Da effective
        'high': int(100 * bins_per_da / 10), # ~100 Da effective
    }

    # Region boundaries in bins
    low_end = int((500 - mz_range[0]) / bin_width_da)
    mid_end = int((1500 - mz_range[0]) / bin_width_da)

    # Compute baseline for each region
    baseline = np.zeros_like(spectrum)

    # Low region
    if low_end > 0:
        baseline[:low_end] = snip(
            spectrum[:low_end],
            max_half_window=windows['low']
        )[0]

    # Mid region
    if mid_end > low_end:
        baseline[low_end:mid_end] = snip(
            spectrum[low_end:mid_end],
            max_half_window=windows['mid']
        )[0]

    # High region
    if n_bins > mid_end:
        baseline[mid_end:] = snip(
            spectrum[mid_end:],
            max_half_window=windows['high']
        )[0]

    # Smooth transitions (avoid discontinuities)
    transition_width = int(50 / bin_width_da)  # 50 Da transition
    baseline = gaussian_filter1d(baseline, sigma=transition_width / 3)

    # Subtract baseline
    corrected = np.maximum(spectrum - baseline, 0)

    return corrected
```

### 3.2 Intensity-Dependent Iterations

```python
def adaptive_snip_iterations(
    spectrum: np.ndarray,
    base_iterations: int = 100,
    max_iterations: int = 200,
) -> np.ndarray:
    """
    Adjust SNIP iterations based on baseline intensity.

    Higher baseline → more iterations needed.
    """
    # Estimate baseline level from lower percentile
    baseline_estimate = np.percentile(spectrum, 10)
    signal_estimate = np.percentile(spectrum, 90)

    if signal_estimate == 0:
        return spectrum

    baseline_ratio = baseline_estimate / signal_estimate

    # Scale iterations: higher baseline → more iterations
    iterations = int(base_iterations * (1 + baseline_ratio * 2))
    iterations = min(iterations, max_iterations)

    return snip(spectrum, max_half_window=iterations)[0]
```

---

## 4. Noise Estimation

### 4.1 MAD-Based Estimation

```python
def estimate_noise_mad(
    spectrum: np.ndarray,
    window_size: int = 500,
) -> np.ndarray:
    """
    Robust noise estimation using Median Absolute Deviation.

    MAD is robust to outliers (peaks) unlike std.
    noise_estimate = MAD * 1.4826 (scaling to Gaussian std)
    """
    from scipy.ndimage import generic_filter

    def mad_filter(x):
        median = np.median(x)
        mad = np.median(np.abs(x - median))
        return mad * 1.4826

    noise_profile = generic_filter(
        spectrum,
        mad_filter,
        size=window_size,
        mode='reflect'
    )

    return np.maximum(noise_profile, 1e-10)
```

### 4.2 Peak-Excluded Noise Estimation

**More accurate when peaks are detectable**

```python
from scipy.signal import find_peaks

def estimate_noise_peak_excluded(
    spectrum: np.ndarray,
    peak_height_factor: float = 3.0,
    window_size: int = 500,
) -> np.ndarray:
    """
    Estimate noise only from peak-free regions.

    1. Detect peaks
    2. Mask peak regions
    3. Estimate noise from non-peak regions
    """
    # Initial noise estimate
    initial_noise = np.std(spectrum) * 0.1

    # Find peaks above noise threshold
    peaks, properties = find_peaks(
        spectrum,
        height=initial_noise * peak_height_factor,
        width=1,
    )

    # Create peak mask (exclude ±5 bins around each peak)
    peak_mask = np.zeros(len(spectrum), dtype=bool)
    for peak_idx in peaks:
        start = max(0, peak_idx - 5)
        end = min(len(spectrum), peak_idx + 6)
        peak_mask[start:end] = True

    # Estimate noise from non-peak regions
    noise_regions = spectrum[~peak_mask]

    if len(noise_regions) < 100:
        # Fallback to MAD if too few non-peak points
        return estimate_noise_mad(spectrum, window_size)

    # Sliding window noise estimate on masked spectrum
    noise_profile = np.zeros_like(spectrum)
    half_window = window_size // 2

    for i in range(len(spectrum)):
        start = max(0, i - half_window)
        end = min(len(spectrum), i + half_window)

        window_mask = ~peak_mask[start:end]
        if window_mask.sum() > 10:
            noise_profile[i] = np.std(spectrum[start:end][window_mask])
        else:
            noise_profile[i] = np.std(spectrum[start:end]) * 0.5

    return np.maximum(noise_profile, 1e-10)
```

---

## 5. MS-Specific Patterns

### 5.1 pyOpenMS Memory-Efficient Loading

```python
from collections.abc import Iterator

from pyopenms import OnDiscMSExperiment, MSSpectrum

def load_spectra_lazy(
    mzml_path: str,
) -> OnDiscMSExperiment:
    """
    Memory-efficient mzML loading.

    OnDiscMSExperiment keeps spectra on disk, loads on demand.
    Essential for large files (>1GB).
    """
    exp = OnDiscMSExperiment()
    exp.openFile(mzml_path)
    return exp


def iterate_ms1_spectra(
    exp: OnDiscMSExperiment,
) -> Iterator[tuple[int, MSSpectrum]]:
    """Iterate over MS1 spectra only."""
    for i in range(exp.getNrSpectra()):
        meta = exp.getMetaData().getSpectrum(i)
        if meta.getMSLevel() == 1:
            spec = exp.getSpectrum(i)
            yield i, spec


def get_spectrum_peaks(
    spectrum: MSSpectrum,
) -> tuple[np.ndarray, np.ndarray]:
    """Extract m/z and intensity arrays from spectrum."""
    mz, intensity = spectrum.get_peaks()
    return np.array(mz), np.array(intensity)
```

### 5.2 Isotope Pattern Handling

```python
def theoretical_isotope_pattern(
    mass: float,
    charge: int = 1,
    n_peaks: int = 10,
    element_composition: str = "averagine",
) -> tuple[np.ndarray, np.ndarray]:
    """
    Generate theoretical isotope pattern.

    Args:
        mass: Monoisotopic mass
        charge: Charge state
        n_peaks: Number of isotope peaks
        element_composition: "averagine" for proteins, or formula string

    Returns:
        mz_values, relative_intensities
    """
    from pyopenms import EmpiricalFormula, CoarseIsotopePatternGenerator

    if element_composition == "averagine":
        # Averagine: average amino acid composition
        # C4.9384 H7.7583 N1.3577 O1.4773 S0.0417
        n_residues = mass / 111.1254  # Average residue mass
        formula = EmpiricalFormula(
            f"C{int(4.9384 * n_residues)}"
            f"H{int(7.7583 * n_residues)}"
            f"N{int(1.3577 * n_residues)}"
            f"O{int(1.4773 * n_residues)}"
            f"S{int(0.0417 * n_residues)}"
        )
    else:
        formula = EmpiricalFormula(element_composition)

    generator = CoarseIsotopePatternGenerator(n_peaks)
    isotopes = generator.run(formula)

    mz_values = np.array([(mass + i * 1.003355) / charge for i in range(n_peaks)])
    intensities = np.array([isotopes.getContainer()[i].getIntensity()
                           for i in range(min(n_peaks, isotopes.size()))])

    # Normalize
    intensities = intensities / intensities.max()

    return mz_values, intensities
```

### 5.3 S/N Validation

```python
def validate_signal_to_noise(
    peak_intensity: float,
    noise_estimate: float,
    min_snr: float = 3.0,
) -> bool:
    """
    Validate peak meets minimum S/N requirement.

    Standard threshold: 3x noise (3-sigma detection).
    """
    if noise_estimate <= 0:
        return False
    snr = peak_intensity / noise_estimate
    return snr >= min_snr


def calculate_snr_spectrum(
    spectrum: np.ndarray,
    noise_profile: np.ndarray,
) -> np.ndarray:
    """Calculate S/N ratio across spectrum."""
    return spectrum / np.maximum(noise_profile, 1e-10)
```

---

## 6. Augmentation Patterns

### 6.1 Calibration Drift Simulation

```python
def augment_calibration_drift(
    mz: np.ndarray,
    ppm_drift: float = 10.0,
    include_quadratic: bool = True,
) -> np.ndarray:
    """
    Simulate m/z calibration drift.

    Real instruments drift ±10-20 ppm over acquisition.
    Includes linear and optionally quadratic components.
    """
    # Linear drift (constant ppm offset)
    linear_drift = np.random.uniform(-ppm_drift, ppm_drift) * 1e-6
    mz_shifted = mz * (1 + linear_drift)

    if include_quadratic:
        # Quadratic drift (m/z-dependent)
        quad_drift = np.random.uniform(-ppm_drift/2, ppm_drift/2) * 1e-6
        mz_shifted += (mz ** 2 / mz.max()) * quad_drift

    return mz_shifted
```

### 6.2 Multi-Component Noise Augmentation

```python
@dataclass
class NoiseParameters:
    """Parameters for realistic MS noise model."""
    shot_noise_scale: float = 1.0
    chemical_noise_peaks: int = 10
    baseline_noise_scale: float = 0.01
    spike_probability: float = 0.001


def augment_realistic_noise(
    spectrum: np.ndarray,
    snr: float = 5.0,
    params: NoiseParameters | None = None,
) -> np.ndarray:
    """
    Add realistic multi-component noise.

    Components:
    1. Shot noise (Poisson-like, intensity-dependent)
    2. Chemical noise (structured peaks from contaminants)
    3. Electronic noise (Gaussian baseline)
    4. Spike noise (rare high-intensity artifacts)
    """
    if params is None:
        params = NoiseParameters()

    noisy = spectrum.copy()
    max_intensity = spectrum.max()

    if max_intensity == 0:
        return noisy

    # 1. Shot noise (intensity-dependent)
    shot_sigma = np.sqrt(np.maximum(spectrum, 0)) * params.shot_noise_scale
    noisy += shot_sigma * np.random.randn(len(spectrum))

    # 2. Electronic baseline noise
    baseline_sigma = max_intensity * params.baseline_noise_scale / snr
    noisy += np.random.normal(0, baseline_sigma, len(spectrum))

    # 3. Chemical noise (structured contaminant peaks)
    n_contaminants = np.random.poisson(params.chemical_noise_peaks)
    for _ in range(n_contaminants):
        center = np.random.randint(0, len(spectrum))
        width = np.random.randint(3, 20)
        intensity = np.random.exponential(max_intensity / snr / 2)

        start = max(0, center - width)
        end = min(len(spectrum), center + width)

        x = np.arange(start, end)
        contaminant = intensity * np.exp(-0.5 * ((x - center) / (width/3)) ** 2)
        noisy[start:end] += contaminant

    # 4. Spike noise (rare artifacts)
    spike_mask = np.random.random(len(spectrum)) < params.spike_probability
    spike_intensities = np.random.exponential(max_intensity * 2, len(spectrum))
    noisy += spike_mask * spike_intensities

    return np.maximum(noisy, 0)
```

---

## 7. Polars Patterns for Spectral Data

### 7.1 Lazy Evaluation for Large Files

```python
import polars as pl

def load_spectral_data_lazy(path: str) -> pl.LazyFrame:
    """
    Lazy loading for large spectral datasets.

    Query optimization happens at collect() time.
    """
    return (
        pl.scan_parquet(path)
        .filter(pl.col("ms_level") == 1)
        .select(["scan_id", "mz", "intensity", "retention_time"])
    )


def aggregate_spectra_by_rt(
    lf: pl.LazyFrame,
    rt_window: float = 0.5,
) -> pl.DataFrame:
    """Aggregate spectra within RT windows."""
    return (
        lf
        .with_columns(
            (pl.col("retention_time") / rt_window).floor().alias("rt_bin")
        )
        .group_by("rt_bin")
        .agg([
            pl.col("mz").flatten(),
            pl.col("intensity").flatten(),
        ])
        .collect(streaming=True)  # Memory-efficient
    )
```

---

## 8. Anti-Patterns to Avoid

### DO NOT:

1. **Use pandas for large spectral data** - Use polars with lazy evaluation
2. **Use fixed binning without considering instrument resolution** - Adapt to data
3. **Use hard-coded region boundaries** - Use smooth transitions
4. **Ignore Poisson nature of count data** - Use appropriate VST
5. **Estimate noise after peak detection** - Estimate before, use for detection
6. **Use sqrt transform for low-count data** - Use Anscombe or GAT
7. **Use histogram binning in production** - Use Gaussian-weighted

### Common Mistakes:

```python
# BAD: Fixed sigma ignoring charge state
binned = gaussian_bin(mz, intensity, sigma=0.5)  # May blur high-z peaks

# GOOD: Adaptive sigma
sigma = estimate_optimal_sigma(mz, intensity, expected_charge_range=(1, 10))
binned = gaussian_bin(mz, intensity, sigma=sigma)

# BAD: Hard boundary in baseline correction
if mz < 500:
    window = 30
else:
    window = 60  # Discontinuity at 500!

# GOOD: Smooth transition
window = interpolate_window(mz, boundaries=[500, 1500], windows=[30, 60, 100])
```

---

## References

- Phase 1 Critical Analysis: Section 1 (Data Preprocessing)
- overall-plan.md: Section 4 (Data Preprocessing Revised)
- pyOpenMS documentation for MS-specific patterns
