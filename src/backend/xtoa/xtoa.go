// xtoa.go implements functions for converting signed integer and floating point numbers into string representations.
// They can be used for printing constant floats and integers at compile time.

package xtoa

// intToChars converts an integer to a byte stream of ASCII characters.
func ItoA(i int) string {
	res := make([]byte, 32) // Signed 64-bit signed int: (2^64) - 1 is ~ 1,9e19 = 20 characters at most.
	var sign bool

	// Check for negative value.
	if i < 0 {
		sign = true
		i = -i
	}

	// Set start index to last index of buffer.
	i1 := len(res) - 1

	// Insert digits back-to-front.
	for ; i1 >= 0 && i != 0; i1-- {
		res[i1] = byte((i % 10) + '0')
		i /= 10
	}

	if sign {
		res[i1] = '-'
		i1--
	}

	return string(res[i1+1:])
}

// FtoA converts a float to a byte stream of ASCII characters with 4 decimal precision.
func FtoA(f float32) string {
	res := make([]byte, 32) // float32 has 4-decimal precision.
	i1 := 0                 // Index of current character being written.

	// Check for negative value.
	if f < 0 {
		f = -f
		res[0] = '-'
		i1++
	}

	ip := int(f)          // Integer part.
	fp := f - float32(ip) // Decimal part.

	// Integer part.
	tmp := ItoA(ip)     // Convert integer part to string.
	copy(res[i1:], tmp) // Copy into result.
	i1 += len(tmp)

	res[i1] = '.' // Add decimal point.
	i1++          // Position of first decimal.

	// Float part.
	fp *= 10000         // 4 decimal precision.
	tmp = ItoA(int(fp)) // Convert decimal part to string.
	copy(res[i1:], tmp)
	i1 += len(tmp)

	return string(res[:i1])
}
