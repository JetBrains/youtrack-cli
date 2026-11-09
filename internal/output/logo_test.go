package output

import "testing"

func TestGradientRGB(t *testing.T) {
	cases := []struct {
		name     string
		c, width int
		want     [3]int
	}{
		{"first column is the start color", 0, 43, gradientFrom},
		{"last column is the end color", 42, 43, gradientTo},
		{"middle column is halfway", 21, 43, [3]int{0xFB, 0x41, 0xB6}},
		{"one-column logo uses the start color", 0, 1, gradientFrom},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gradientRGB(tc.c, tc.width); got != tc.want {
				t.Errorf("gradientRGB(%d, %d) = %#v, want %#v", tc.c, tc.width, got, tc.want)
			}
		})
	}
}
