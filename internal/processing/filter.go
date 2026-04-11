package processing

import (
	"context"
	"fmt"
	"math"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("filter", func(id string) pipeline.Block {
		return &filterBlock{
			id:           id,
			mode:         "lowpass",
			cutoffHz:     100,
			order:        2,
			sampleRateHz: 1000,
		}
	})
}

type filterBlock struct {
	mu           sync.RWMutex
	id           string
	mode         string
	cutoffHz     float64
	cutoffLowHz  float64
	cutoffHighHz float64
	order        int
	sampleRateHz float64
	coeffB       []float64
	coeffA       []float64
	state        []float64 // Direct Form II Transposed delay line
}

func (f *filterBlock) ID() string                       { return f.id }
func (f *filterBlock) Type() string                     { return "filter" }
func (f *filterBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (f *filterBlock) Configure(params map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	mode := f.mode
	if v, ok := params["mode"].(string); ok {
		mode = v
	}
	switch mode {
	case "lowpass", "highpass", "bandpass", "bandstop":
	default:
		return fmt.Errorf("invalid filter mode: %s", mode)
	}

	cutoffHz := f.cutoffHz
	if v, ok := params["cutoffHz"]; ok {
		cutoffHz = toFloat(v, f.cutoffHz)
	}
	cutoffLowHz := f.cutoffLowHz
	if v, ok := params["cutoffLowHz"]; ok {
		cutoffLowHz = toFloat(v, f.cutoffLowHz)
	}
	cutoffHighHz := f.cutoffHighHz
	if v, ok := params["cutoffHighHz"]; ok {
		cutoffHighHz = toFloat(v, f.cutoffHighHz)
	}
	order := f.order
	if v, ok := params["order"]; ok {
		order = toIntP(v, f.order)
	}
	sampleRateHz := f.sampleRateHz
	if v, ok := params["sampleRateHz"]; ok {
		sampleRateHz = toFloat(v, f.sampleRateHz)
	}

	if order < 1 || order > 8 {
		return fmt.Errorf("filter order must be 1-8, got %d", order)
	}
	if sampleRateHz <= 0 {
		return fmt.Errorf("sampleRateHz must be > 0")
	}

	nyquist := sampleRateHz / 2
	switch mode {
	case "lowpass", "highpass":
		if cutoffHz <= 0 || cutoffHz >= nyquist {
			return fmt.Errorf("cutoffHz must be > 0 and < Nyquist (%f), got %f", nyquist, cutoffHz)
		}
	case "bandpass", "bandstop":
		if cutoffLowHz <= 0 || cutoffHighHz <= 0 {
			return fmt.Errorf("cutoffLowHz and cutoffHighHz must be > 0")
		}
		if cutoffLowHz >= cutoffHighHz {
			return fmt.Errorf("cutoffLowHz (%f) must be < cutoffHighHz (%f)", cutoffLowHz, cutoffHighHz)
		}
		if cutoffHighHz >= nyquist {
			return fmt.Errorf("cutoffHighHz must be < Nyquist (%f)", nyquist)
		}
	}

	f.mode = mode
	f.cutoffHz = cutoffHz
	f.cutoffLowHz = cutoffLowHz
	f.cutoffHighHz = cutoffHighHz
	f.order = order
	f.sampleRateHz = sampleRateHz

	b, a := butterworthCoeffs(mode, cutoffHz, cutoffLowHz, cutoffHighHz, order, sampleRateHz)
	f.coeffB = b
	f.coeffA = a
	f.state = make([]float64, len(b)-1)

	return nil
}

func (f *filterBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (f *filterBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (f *filterBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	_ chan<- pipeline.BlockError,
) error {
	in, ok := inputs["in"]
	if !ok {
		return nil
	}
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, open := <-in:
			if !open {
				return nil
			}
			if len(chunk.Values) == 0 {
				continue
			}

			f.mu.RLock()
			b := f.coeffB
			a := f.coeffA
			st := f.state
			f.mu.RUnlock()

			if len(b) == 0 || len(a) == 0 {
				continue
			}

			// Apply Direct Form II Transposed IIR filter to each value
			filtered := make([]float64, len(chunk.Values))
			for i, x := range chunk.Values {
				y := b[0]*x + st[0]
				for j := 0; j < len(st)-1; j++ {
					st[j] = b[j+1]*x - a[j+1]*y + st[j+1]
				}
				st[len(st)-1] = b[len(b)-1]*x - a[len(a)-1]*y
				filtered[i] = y
			}

			f.mu.Lock()
			f.state = st
			f.mu.Unlock()

			out <- pipeline.DataChunk{
				Timestamp: chunk.Timestamp,
				SourceID:  chunk.SourceID,
				Values:    filtered,
			}
		}
	}
}

// --- Butterworth coefficient computation via bilinear transform ---

// butterworthCoeffs computes IIR filter coefficients for the given mode and parameters.
// Returns (b, a) where a[0] is always 1.0 (normalized).
func butterworthCoeffs(mode string, cutoffHz, cutoffLowHz, cutoffHighHz float64, order int, sampleRateHz float64) ([]float64, []float64) {
	switch mode {
	case "lowpass":
		return butterworthLP(order, cutoffHz, sampleRateHz)
	case "highpass":
		return butterworthHP(order, cutoffHz, sampleRateHz)
	case "bandpass":
		return butterworthBP(order, cutoffLowHz, cutoffHighHz, sampleRateHz)
	case "bandstop":
		return butterworthBS(order, cutoffLowHz, cutoffHighHz, sampleRateHz)
	default:
		return []float64{1}, []float64{1}
	}
}

// butterworthSections returns the pole angles for cascaded second-order sections.
// For order N: floor(N/2) conjugate pairs + 1 real pole if N is odd.
// Returns (pairAngles []float64, hasOddPole bool).
func butterworthSections(order int) ([]float64, bool) {
	nPairs := order / 2
	angles := make([]float64, nPairs)
	for i := 0; i < nPairs; i++ {
		angles[i] = math.Pi * (2*float64(i) + 1) / (2 * float64(order))
	}
	return angles, order%2 == 1
}

// butterworthLP computes lowpass Butterworth coefficients using cascaded second-order sections.
func butterworthLP(order int, fc, fs float64) ([]float64, []float64) {
	wc := math.Tan(math.Pi * fc / fs)
	wc2 := wc * wc
	b := []float64{1}
	a := []float64{1}

	angles, hasOdd := butterworthSections(order)
	for _, theta := range angles {
		sinT := math.Sin(theta)
		d := 1 + 2*wc*sinT + wc2
		bk := []float64{wc2 / d, 2 * wc2 / d, wc2 / d}
		ak := []float64{1, 2 * (wc2 - 1) / d, (1 - 2*wc*sinT + wc2) / d}
		b = convolve(b, bk)
		a = convolve(a, ak)
	}
	if hasOdd {
		a0 := wc + 1
		bk := []float64{wc / a0, wc / a0}
		ak := []float64{1, (wc - 1) / a0}
		b = convolve(b, bk)
		a = convolve(a, ak)
	}

	return b, a
}

// butterworthHP computes highpass Butterworth coefficients.
func butterworthHP(order int, fc, fs float64) ([]float64, []float64) {
	wc := math.Tan(math.Pi * fc / fs)
	wc2 := wc * wc
	b := []float64{1}
	a := []float64{1}

	angles, hasOdd := butterworthSections(order)
	for _, theta := range angles {
		sinT := math.Sin(theta)
		d := 1 + 2*wc*sinT + wc2
		bk := []float64{1 / d, -2 / d, 1 / d}
		ak := []float64{1, 2 * (wc2 - 1) / d, (1 - 2*wc*sinT + wc2) / d}
		b = convolve(b, bk)
		a = convolve(a, ak)
	}
	if hasOdd {
		a0 := wc + 1
		bk := []float64{1 / a0, -1 / a0}
		ak := []float64{1, (wc - 1) / a0}
		b = convolve(b, bk)
		a = convolve(a, ak)
	}

	return b, a
}

// butterworthBP computes bandpass Butterworth coefficients.
// Order N LP prototype → order 2N bandpass.
// Strategy: design Nth-order LP at normalized cutoff, then apply LP-to-BP spectral transform.
func butterworthBP(order int, fLow, fHigh, fs float64) ([]float64, []float64) {
	// First design a lowpass prototype, then transform to bandpass
	// Compute center frequency and bandwidth in pre-warped domain
	wLow := math.Tan(math.Pi * fLow / fs)
	wHigh := math.Tan(math.Pi * fHigh / fs)
	w0 := math.Sqrt(wLow * wHigh) // geometric center
	bw := wHigh - wLow

	// Get LP prototype coefficients at normalized cutoff = 1
	bLP, aLP := butterworthLP(order, fs/(2*math.Pi), fs) // cutoff = fs/(2π) → wc = tan(π·fc/fs) = tan(0.5) ≈ ...

	// Instead of transforming coefficients, use a direct approach:
	// For each LP prototype pole, compute two BP poles via the LP→BP transform
	// s_bp = (s_lp² + w0²) / (s_lp * BW)
	// This maps each LP pole to two BP poles.

	// Simpler: cascade second-order BP sections directly.
	// Each LP conjugate pair at angle θ → two 2nd-order BP digital sections
	_ = bLP
	_ = aLP

	b := []float64{1}
	a := []float64{1}

	angles, hasOdd := butterworthSections(order)
	for _, theta := range angles {
		sinT := math.Sin(theta)
		// Each conjugate pair maps to two 2nd-order BP sections
		// The LP prototype has poles at: p = -sin(θ) ± j·cos(θ) (unit circle)
		// LP-to-BP transform: each LP pole p → two BP poles via quadratic
		// s = (p·BW ± sqrt((p·BW)² - 4·w0²)) / 2

		// The two resulting second-order sections have:
		sigma := sinT * bw / 2
		omega2 := w0 * w0

		// Section 1 & 2 from solving the quadratic for the BP pole locations
		// After bilinear transform, each section is a standard resonator
		// Use two cascaded 2nd-order sections with the same resonant behavior
		// but different Q factors based on the original pole location

		// Direct approach: compute 4th-order section coefficients
		// from the LP-to-BP transformed analog transfer function, then bilinear
		s := sigma
		w2 := omega2
		// Analog BP denominator from one LP conjugate pair:
		// (s² + 2σs + σ² + cos²(θ)·bw²/4 + w0²)  ... this gets messy.

		// Cleanest: use the bilinear-transformed cookbook formulas for two cascaded resonators
		// Each LP conjugate pair at angle θ with damping ζ = sin(θ)
		// maps to center frequency w0 with two Q values

		// Actually, let's just cascade the LP section with the LP-to-BP z-domain transform.
		// More practical: compute the 4th-order section directly.

		// For a 2nd-order LP section H_lp(s) = wc² / (s² + 2·ζ·wc·s + wc²)
		// Apply LP→BP: s → (s² + w0²)/(s·BW)
		// Result: a 4th-order BP transfer function

		// After bilinear transform s = 2·fs·(z-1)/(z+1):
		// This produces a 4th-order digital section with 5 coefficients

		// Let me use the standard z-domain LP-to-BP transform approach:
		// Given digital LP H(z) with coefficients, apply the allpass transform
		// z⁻¹ → -(z⁻² - α₁·z⁻¹ + α₂) / (α₂·z⁻² - α₁·z⁻¹ + 1)
		// This doubles the filter order.

		// For simplicity and correctness, I'll compute analog BP poles directly
		// and apply bilinear transform to each pole pair.

		cosT := math.Cos(theta)
		_ = s
		_ = w2

		// Analog LP prototype pole (normalized): p = -sinT + j·cosT
		// LP→BP transform: s_lp → (s² + w0²) / (s · BW)
		// Solving: s · BW · p = s² + w0² → s² - BW·p·s + w0² = 0
		// s = (BW·p ± sqrt(BW²·p² - 4·w0²)) / 2

		// p = -sinT + j·cosT, p² = sin²T - cos²T - 2j·sinT·cosT = -cos(2θ) - j·sin(2θ)
		// BW·p = BW·(-sinT + j·cosT)
		// (BW·p)² = BW²·(sin²T - cos²T - 2j·sinT·cosT) = BW²·(-cos2θ - j·sin2θ)

		bwp_r := bw * (-sinT)
		bwp_i := bw * cosT
		bwp2_r := bwp_r*bwp_r - bwp_i*bwp_i
		bwp2_i := 2 * bwp_r * bwp_i
		disc_r := bwp2_r - 4*w0*w0
		disc_i := bwp2_i

		// sqrt of complex discriminant
		disc_mag := math.Sqrt(disc_r*disc_r + disc_i*disc_i)
		disc_ang := math.Atan2(disc_i, disc_r)
		sqrt_r := math.Sqrt(disc_mag) * math.Cos(disc_ang/2)
		sqrt_i := math.Sqrt(disc_mag) * math.Sin(disc_ang/2)

		// Two analog BP poles: s1 = (BW·p + sqrt) / 2, s2 = (BW·p - sqrt) / 2
		s1_r := (bwp_r + sqrt_r) / 2
		s1_i := (bwp_i + sqrt_i) / 2
		s2_r := (bwp_r - sqrt_r) / 2
		s2_i := (bwp_i - sqrt_i) / 2

		// Each analog pole s = σ + jω with its conjugate s* = σ - jω
		// forms a 2nd-order section: (s - pole)(s - pole*) = s² - 2σs + (σ² + ω²)
		// After bilinear transform s = (z-1)/(z+1) (pre-warping already done):

		// Section from s1 and s1*: analog denominator = s² - 2·s1_r·s + (s1_r² + s1_i²)
		_ = sigma
		bpSection := func(sr, si float64) ([]float64, []float64) {
			// Analog: H(s) = s / (s² + (-2·sr)·s + (sr²+si²))
			// but for bandpass numerator we need to track properly.
			// Actually for Butterworth BP, the overall numerator has the form s^N (for order N LP → 2N BP)
			// So each section contributes an s factor to the numerator.
			mag2 := sr*sr + si*si
			twoSr := 2 * sr

			// Apply bilinear s=(z-1)/(z+1), multiply top and bottom by (z+1)²:
			// Denominator: (z-1)² - 2·sr·(z-1)(z+1) + mag2·(z+1)²
			//            = z² - 2z + 1 - 2sr·(z²-1) + mag2·(z²+2z+1)
			//            = (1-2sr+mag2)z² + (-2+2·mag2)z + (1+2sr+mag2)
			// Wait, need to be careful: bilinear s = 2fs·(z-1)/(z+1), but since we pre-warped,
			// we use s = (z-1)/(z+1) (without 2fs factor, as the analog frequencies are already warped)

			a0 := 1 - twoSr + mag2
			a1 := -2 + 2*mag2
			a2 := 1 + twoSr + mag2

			// Numerator: s → (z-1)/(z+1), contributes one (z-1)/(z+1) per s in numerator
			// For bandpass, each section's numerator contribution is proportional to s
			// s = (z-1)/(z+1) → numerator (z-1), denominator gets extra (z+1)
			// But the overall gain normalization handles this.

			// Normalized 2nd-order section:
			// H(z) = (z² - 1) / (a0·z² + a1·z + a2)  [from s/(s²+...)]
			// Actually: s / den(s) → (z-1)(z+1) / ((z+1)² · den_digital)... this is getting complex.

			// Let me just do it directly.
			// Analog transfer function for this section: H_a(s) = s / (s² - 2·sr·s + mag2)
			// s = (z-1)/(z+1):
			// Num: (z-1)/(z+1)
			// Den: ((z-1)/(z+1))² - 2sr·(z-1)/(z+1) + mag2
			//    = (z-1)²/(z+1)² - 2sr·(z-1)/(z+1) + mag2
			// Multiply through by (z+1)²:
			// NumZ = (z-1)·(z+1) = z² - 1
			// DenZ = (z-1)² - 2sr·(z-1)(z+1) + mag2·(z+1)²
			//      = z²-2z+1 - 2sr·(z²-1) + mag2·(z²+2z+1)
			//      = (1-2sr+mag2)z² + (-2+2mag2)z + (1+2sr+mag2)

			nb := []float64{1 / a0, 0, -1 / a0}
			na := []float64{1, a1 / a0, a2 / a0}
			return nb, na
		}

		bk1, ak1 := bpSection(s1_r, s1_i)
		bk2, ak2 := bpSection(s2_r, s2_i)
		b = convolve(b, bk1)
		a = convolve(a, ak1)
		b = convolve(b, bk2)
		a = convolve(a, ak2)
	}
	if hasOdd {
		// Single real LP pole at s = -1 (normalized)
		// LP→BP: s² + BW·s + w0² = 0 → s = (-BW ± sqrt(BW²-4w0²))/2
		disc := bw*bw - 4*w0*w0
		if disc >= 0 {
			// Two real poles
			sq := math.Sqrt(disc)
			s1 := (-bw + sq) / 2
			s2 := (-bw - sq) / 2
			// Each real pole: H(s) = 1/(s - s_k) → bilinear → first-order section
			// But for BP numerator, we have s → (z²-1)/(z+1)²
			// Actually, combined the two: analog = s / (s² + BW·s + w0²)
			// Bilinear: num = z²-1, den = (1+BW+w0²) z² + 2(w0²-1) z + (1-BW+w0²)
			_ = s1
			_ = s2
		}
		// Direct computation for the odd pole:
		// Analog section: H(s) = s / (s² + bw·s + w0²)
		// After bilinear s=(z-1)/(z+1):
		den := 1 + bw + w0*w0
		bk := []float64{1 / den, 0, -1 / den}
		ak := []float64{1, 2 * (w0*w0 - 1) / den, (1 - bw + w0*w0) / den}
		b = convolve(b, bk)
		a = convolve(a, ak)
	}

	// Normalize gain at center frequency
	// Evaluate H(z) at z = e^(j·2π·fc/fs) where fc = sqrt(fLow·fHigh)
	fc := math.Sqrt(fLow * fHigh)
	w := 2 * math.Pi * fc / fs
	numR, numI := evalPoly(b, w)
	denR, denI := evalPoly(a, w)
	numMag := math.Sqrt(numR*numR + numI*numI)
	denMag := math.Sqrt(denR*denR + denI*denI)
	if numMag > 1e-15 {
		gain := denMag / numMag
		for i := range b {
			b[i] *= gain
		}
	}

	return b, a
}

// evalPoly evaluates polynomial p at z = e^(j·w) and returns (real, imag).
func evalPoly(p []float64, w float64) (float64, float64) {
	re, im := 0.0, 0.0
	for k, c := range p {
		angle := -float64(k) * w
		re += c * math.Cos(angle)
		im += c * math.Sin(angle)
	}
	return re, im
}

// butterworthBS computes bandstop (notch) Butterworth coefficients.
// Order N LP prototype → order 2N bandstop.
func butterworthBS(order int, fLow, fHigh, fs float64) ([]float64, []float64) {
	wLow := math.Tan(math.Pi * fLow / fs)
	wHigh := math.Tan(math.Pi * fHigh / fs)
	w0 := math.Sqrt(wLow * wHigh)
	bw := wHigh - wLow

	b := []float64{1}
	a := []float64{1}

	angles, hasOdd := butterworthSections(order)
	for _, theta := range angles {
		sinT := math.Sin(theta)
		cosT := math.Cos(theta)

		// Same pole computation as BP
		bwp_r := bw * (-sinT)
		bwp_i := bw * cosT
		bwp2_r := bwp_r*bwp_r - bwp_i*bwp_i
		bwp2_i := 2 * bwp_r * bwp_i
		disc_r := bwp2_r - 4*w0*w0
		disc_i := bwp2_i

		disc_mag := math.Sqrt(disc_r*disc_r + disc_i*disc_i)
		disc_ang := math.Atan2(disc_i, disc_r)
		sqrt_r := math.Sqrt(disc_mag) * math.Cos(disc_ang/2)
		sqrt_i := math.Sqrt(disc_mag) * math.Sin(disc_ang/2)

		s1_r := (bwp_r + sqrt_r) / 2
		s1_i := (bwp_i + sqrt_i) / 2
		s2_r := (bwp_r - sqrt_r) / 2
		s2_i := (bwp_i - sqrt_i) / 2

		// For bandstop, numerator has (s²+w0²) factors instead of s factors
		// Analog section: (s²+w0²) / (s² - 2·sr·s + (sr²+si²))
		bsSection := func(sr, si float64) ([]float64, []float64) {
			mag2 := sr*sr + si*si
			// Den after bilinear:
			a0 := 1 - 2*sr + mag2
			a1 := -2 + 2*mag2
			a2 := 1 + 2*sr + mag2

			// Num: s²+w0² = ((z-1)/(z+1))² + w0²
			// Multiply by (z+1)²: (z-1)² + w0²·(z+1)² = (1+w0²)z² + (-2+2w0²)z + (1+w0²)
			w02 := w0 * w0
			n0 := (1 + w02) / a0
			n1 := (-2 + 2*w02) / a0
			n2 := (1 + w02) / a0

			nb := []float64{n0, n1, n2}
			na := []float64{1, a1 / a0, a2 / a0}
			return nb, na
		}

		bk1, ak1 := bsSection(s1_r, s1_i)
		bk2, ak2 := bsSection(s2_r, s2_i)
		b = convolve(b, bk1)
		a = convolve(a, ak1)
		b = convolve(b, bk2)
		a = convolve(a, ak2)
	}
	if hasOdd {
		// Analog section: (s²+w0²) / (s² + bw·s + w0²)
		// After bilinear: num = (1+w0²)z² + 2(w0²-1)z + (1+w0²)
		// den = (1+bw+w0²)z² + 2(w0²-1)z + (1-bw+w0²)
		w02 := w0 * w0
		den := 1 + bw + w02
		bk := []float64{(1 + w02) / den, 2 * (w02 - 1) / den, (1 + w02) / den}
		ak := []float64{1, 2 * (w02 - 1) / den, (1 - bw + w02) / den}
		b = convolve(b, bk)
		a = convolve(a, ak)
	}

	return b, a
}

// convolve computes the convolution of two coefficient slices.
func convolve(a, b []float64) []float64 {
	n := len(a) + len(b) - 1
	result := make([]float64, n)
	for i, ai := range a {
		for j, bj := range b {
			result[i+j] += ai * bj
		}
	}
	return result
}
