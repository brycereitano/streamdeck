package streamdeck

import (
	"bytes"
	"image"
	"io"
)

// User Logic Constants
const (
	numRows    = 3
	numColumns = 5
	numButtons = 15
	iconSize   = 72
)

// USB Protocol Constants
const (
	numFirstPagePixels  = 2583
	numSecondPagePixels = 2601

	pageSize  = 8191
	eventSize = 17
)

type Device interface {
	io.ReadWriteCloser
}

type Client struct {
	device Device
}

func New(device Device) (*Client, error) {
	return &Client{device: device}, nil
}

func (c *Client) Buttons() ([]bool, error) {
	buttons := make([]bool, numButtons)

	var buf = make([]byte, eventSize)
	_, err := c.device.Read(buf)
	if err != nil {
		return nil, err
	}
	for i := 0; i < numButtons; i++ {
		buttons[i] = buf[i+1] == 1
	}

	return buttons, nil
}

func (c *Client) SetPanelImage(img image.Image) error {
	subImg := img.(subImager)
	for column := 0; column < numColumns; column++ {
		for row := 0; row < numRows; row++ {
			startX := (numColumns - 1 - column) * iconSize
			startY := row * iconSize
			crp := subImg.SubImage(image.Rect(startX, startY, startX+iconSize, startY+iconSize))
			err := c.SetKeyImage(column+row*numColumns, crp)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) ClearPanel() error {
	for i := 0; i < numButtons; i++ {
		if err := c.ClearKey(i); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) SetKeyImage(key int, img image.Image) error {
	pixels := make([]byte, iconSize*iconSize*3)
	min := img.Bounds().Min
	max := img.Bounds().Max
	for x := min.X; x < max.X; x++ {
		for y := min.Y; y < max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			i := ((y-min.Y)*iconSize + (iconSize - (x - min.X))) * 3
			pixels[i-1] = uint8(r >> 8)
			pixels[i-2] = uint8(g >> 8)
			pixels[i-3] = uint8(b >> 8)
		}
	}

	err := c.writePage1(key, pixels[:numFirstPagePixels*3])
	if err != nil {
		return err
	}

	return c.writePage2(key, pixels[numFirstPagePixels*3:])
}

func (c *Client) SetKeyColor(keyIndex int, r byte, g byte, b byte) error {
	pixel := []byte{b, g, r}

	page1Bytes := bytes.Repeat(pixel, 3*numFirstPagePixels)
	err := c.writePage1(keyIndex, page1Bytes)
	if err != nil {
		return err
	}

	page2Bytes := bytes.Repeat(pixel, 3*numSecondPagePixels)
	return c.writePage2(keyIndex, page2Bytes)
}

func (c *Client) ClearKey(keyIndex int) error {
	page1Bytes := make([]byte, 3*3*numFirstPagePixels)
	err := c.writePage1(keyIndex, page1Bytes)
	if err != nil {
		return err
	}

	page2Bytes := make([]byte, 3*3*numSecondPagePixels)
	return c.writePage2(keyIndex, page2Bytes)
}

func (c *Client) writePage1(key int, buf []byte) error {
	var headerPage1 = []byte{
		0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x42, 0x4d, 0xf6, 0x3c, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x48, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xc0, 0x3c, 0x00, 0x00, 0xc4, 0x0e,
		0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	page := make([]byte, pageSize)
	copy(page, headerPage1)
	copy(page[len(headerPage1):], buf)
	page[5] = byte(key + 1)

	_, err := c.device.Write(page)
	return err
}

func (c *Client) writePage2(key int, buf []byte) error {
	var headerPage2 = []byte{
		0x02, 0x01, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	page := make([]byte, pageSize)
	copy(page, headerPage2)
	copy(page[len(headerPage2):], buf)
	page[5] = byte(key + 1)

	_, err := c.device.Write(page)
	return err
}

func (c *Client) Close() error {
	return c.device.Close()
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}
