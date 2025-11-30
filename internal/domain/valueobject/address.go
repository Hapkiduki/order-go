// Package valueobject contains value objects that represent concepts without identity.
package valueobject

import "fmt"

// Dimensions represents the physical dimensions for shipping calculations.
// All measurements are in centimeters.
type Dimensions struct {
	// Length in centimeters.
	Length float64 `json:"length"`

	// Width in centimeters.
	Width float64 `json:"width"`

	// Height in centimeters.
	Height float64 `json:"height"`
}

// NewDimensions creates a new Dimensions value object.
//
// Parameters:
//   - length: Length in centimeters
//   - width: Width in centimeters
//   - height: Height in centimeters
//
// Returns:
//   - Dimensions: new Dimensions value object
func NewDimensions(length, width, height float64) Dimensions {
	return Dimensions{
		Length: length,
		Width:  width,
		Height: height,
	}
}

// Volume calculates the volume in cubic centimeters.
//
// Returns:
//   - float64: volume in cm³
func (d Dimensions) Volume() float64 {
	return d.Length * d.Width * d.Height
}

// VolumetricWeight calculates the volumetric weight for shipping.
// Uses DIM factor of 5000 (standard for international shipping).
//
// Returns:
//   - float64: volumetric weight in kg
func (d Dimensions) VolumetricWeight() float64 {
	return d.Volume() / 5000
}

// IsEmpty checks if all dimensions are zero.
//
// Returns:
//   - bool: true if all dimensions are zero
func (d Dimensions) IsEmpty() bool {
	return d.Length == 0 && d.Width == 0 && d.Height == 0
}

// String returns a formatted string representation.
//
// Returns:
//   - string: formatted dimensions (e.g., "30x20x10 cm")
func (d Dimensions) String() string {
	return fmt.Sprintf("%.1fx%.1fx%.1f cm", d.Length, d.Width, d.Height)
}
