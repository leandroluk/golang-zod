# Example: Bidirectional Codecs

This recipe demonstrates using `z.Codec` to convert complex data structures back and forth with symmetry guarantees.

## Complete Example

```go
package main

import (
	"fmt"
	"strings"

	z "github.com/leandroluk/golang-zod"
)

// Define target domain type
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func main() {
	// String to Coordinates Codec ("lat,lng" <-> Coordinates)
	coordCodec := z.Codec[string, Coordinates](
		z.String(),
		nil,
		// Decode: parse "40.7128,-74.0060" into Coordinates
		func(raw string) (Coordinates, error) {
			var lat, lng float64
			_, err := fmt.Sscanf(strings.TrimSpace(raw), "%f,%f", &lat, &lng)
			if err != nil {
				return Coordinates{}, fmt.Errorf("invalid coordinate pair %q (expected 'lat,lng')", raw)
			}
			return Coordinates{Latitude: lat, Longitude: lng}, nil
		},
		// Encode: format Coordinates back into "lat,lng" string
		func(c Coordinates) (string, error) {
			return fmt.Sprintf("%.4f,%.4f", c.Latitude, c.Longitude), nil
		},
	)

	// 1. Decoding
	coords, err := coordCodec.DecodeValue("37.7749,-122.4194")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Decoded Coordinates: Lat=%.4f, Lng=%.4f\n", coords.Latitude, coords.Longitude)

	// 2. Encoding
	encoded, err := coordCodec.Encode(coords)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Encoded String: %s\n", encoded)
}
```
