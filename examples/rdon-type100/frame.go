package main

import (
	"fmt"

	"github.com/rdon-key/modgadget"
)

type uiRect struct {
	x, y, width, height int16
}

var menuFrameBounds = uiRect{x: menuFrameX, y: menuFrameY, width: menuFrameWidth, height: menuFrameHeight}

var inputFrameBounds = uiRect{x: inputFrameX, y: inputFrameY, width: inputFrameWidth, height: inputFrameHeight}

var inputInteriorBounds = uiRect{x: inputFrameX + 1, y: inputFrameY + 1, width: inputFrameWidth - 2, height: inputFrameHeight - 2}

func drawFrame(target modgadget.Display, bounds uiRect, color modgadget.Color565, scratch *[64]byte) error {
	if target == nil || scratch == nil || bounds.width < 2 || bounds.height < 2 {
		return fmt.Errorf("rdon-type100: invalid frame target or bounds")
	}
	edges := [...]uiRect{
		{x: bounds.x, y: bounds.y, width: bounds.width, height: 1},
		{x: bounds.x, y: bounds.y + bounds.height - 1, width: bounds.width, height: 1},
		{x: bounds.x, y: bounds.y + 1, width: 1, height: bounds.height - 2},
		{x: bounds.x + bounds.width - 1, y: bounds.y + 1, width: 1, height: bounds.height - 2},
	}
	for _, edge := range edges {
		if err := fillSolidRect(target, edge, color, scratch); err != nil {
			return fmt.Errorf("rdon-type100: draw menu frame: %w", err)
		}
	}
	return nil
}

func fillSolidRect(target modgadget.Display, bounds uiRect, color modgadget.Color565, scratch *[64]byte) error {
	hi, lo := byte(uint16(color)>>8), byte(color)
	for index := 0; index < len(scratch); index += 2 {
		scratch[index], scratch[index+1] = hi, lo
	}
	if err := target.BeginRect(bounds.x, bounds.y, bounds.width, bounds.height); err != nil {
		return err
	}
	remaining := int(bounds.width) * int(bounds.height) * 2
	for remaining > 0 {
		count := remaining
		if count > len(scratch) {
			count = len(scratch)
		}
		if err := target.WritePixels(scratch[:count]); err != nil {
			return err
		}
		remaining -= count
	}
	return target.EndRect()
}
