package display

// FillRect fills a rectangle using a reusable RGB565 transfer buffer.
func FillRect(backend Backend, rect Rect, color Color565, scratch []byte) error {
	if backend == nil {
		return ErrNilBackend
	}
	if rect.Empty() {
		return ErrInvalidRect
	}
	usable := len(scratch) &^ 1
	if usable < 2 {
		return ErrScratchTooSmall
	}

	high := byte(color >> 8)
	low := byte(color)
	for i := 0; i < usable; i += 2 {
		scratch[i] = high
		scratch[i+1] = low
	}

	remaining := int64(rect.PixelCount()) * 2
	if err := backend.BeginRect(rect.X, rect.Y, rect.Width, rect.Height); err != nil {
		return err
	}
	for remaining > 0 {
		n := usable
		if int64(n) > remaining {
			n = int(remaining)
		}
		if err := backend.WritePixels(scratch[:n]); err != nil {
			return err
		}
		remaining -= int64(n)
	}
	return backend.EndRect()
}

// BlitRGB565 streams a stride-based big-endian RGB565 image into a rectangle.
func BlitRGB565(backend Backend, dst Rect, pixels []byte, stride int) error {
	if backend == nil {
		return ErrNilBackend
	}
	if dst.Empty() {
		return ErrInvalidRect
	}

	rowBytes := int(dst.Width) * 2
	if stride < rowBytes {
		return ErrInvalidStride
	}
	if len(pixels) < rowBytes {
		return ErrPixelDataTooShort
	}
	rowGaps := int(dst.Height) - 1
	if rowGaps > 0 && stride > (len(pixels)-rowBytes)/rowGaps {
		return ErrPixelDataTooShort
	}

	if err := backend.BeginRect(dst.X, dst.Y, dst.Width, dst.Height); err != nil {
		return err
	}
	for row := 0; row < int(dst.Height); row++ {
		start := row * stride
		if err := backend.WritePixels(pixels[start : start+rowBytes]); err != nil {
			return err
		}
	}
	return backend.EndRect()
}
