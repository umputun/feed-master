package mp3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

type (
	// Decoder translates a io.Reader into a series of frames
	Decoder struct {
		src io.Reader
		err error
	}

	// Frame represents one individual mp3 frame
	Frame struct {
		buf []byte
	}

	// FrameHeader represents the entire header of a frame
	FrameHeader []byte

	// FrameVersion is the MPEG version given in the frame header
	FrameVersion byte
	// FrameLayer is the MPEG layer given in the frame header
	FrameLayer byte
	// FrameEmphasis is the Emphasis value from the frame header
	FrameEmphasis byte
	// FrameChannelMode is the Channel mode from the frame header
	FrameChannelMode byte
	// FrameBitRate is the bit rate from the frame header
	FrameBitRate int
	// FrameSampleRate is the sample rate from teh frame header
	FrameSampleRate int

	// FrameSideInfo holds the SideInfo bytes from the frame
	FrameSideInfo []byte
)

//go:generate stringer -type=FrameVersion
const (
	MPEG25 FrameVersion = iota
	MPEGReserved
	MPEG2
	MPEG1
	VERSIONMAX
)

//go:generate stringer -type=FrameLayer
const (
	LayerReserved FrameLayer = iota
	Layer3
	Layer2
	Layer1
	LayerMax
)

//go:generate stringer -type=FrameEmphasis
const (
	EmphNone FrameEmphasis = iota
	Emph5015
	EmphReserved
	EmphCCITJ17
	EmphMax
)

//go:generate stringer -type=FrameChannelMode
const (
	Stereo FrameChannelMode = iota
	JointStereo
	DualChannel
	SingleChannel
	ChannelModeMax
)

const (
	// ErrInvalidBitrate indicates that the header information did not contain a recognized bitrate
	ErrInvalidBitrate FrameBitRate = -1
)

var (
	bitrates = [VERSIONMAX][LayerMax][15]int{
		{ // MPEG 2.5
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                       // LayerReserved
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer3
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer2
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256}, // Layer1
		},
		{ // Reserved
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // LayerReserved
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Layer3
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Layer2
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Layer1
		},
		{ // MPEG 2
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                       // LayerReserved
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer3
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer2
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256}, // Layer1
		},
		{ // MPEG 1
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                          // LayerReserved
			{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},     // Layer3
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},    // Layer2
			{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448}, // Layer1
		},
	}
	sampleRates = [int(VERSIONMAX)][3]int{
		{11025, 12000, 8000},  //MPEG25
		{0, 0, 0},             //MPEGReserved
		{22050, 24000, 16000}, //MPEG2
		{44100, 48000, 32000}, //MPEG1
	}

	// ErrInvalidSampleRate indicates that no samplerate could be found for the frame header provided
	ErrInvalidSampleRate = FrameSampleRate(-1)

	samplesPerFrame = [VERSIONMAX][LayerMax]int{
		{ // MPEG25
			0,
			576,
			1152,
			384,
		},
		{ // Reserved
			0,
			0,
			0,
			0,
		},
		{ // MPEG2
			0,
			576,
			1152,
			384,
		},
		{ // MPEG1
			0,
			1152,
			1152,
			384,
		},
	}
	slotSize = [LayerMax]int{
		0, //	LayerReserved
		1, //	Layer3
		1, //	Layer2
		4, //	Layer1
	}

	// ErrNoSyncBits implies we could not find a valid frame header sync bit before EOF
	ErrNoSyncBits = errors.New("EOF before sync bits found")

	// ErrPrematureEOF indicates that the filed ended before a complete frame could be read
	ErrPrematureEOF = errors.New("EOF mid stream")
)

func init() {
	bitrates[MPEG25] = bitrates[MPEG2]
	samplesPerFrame[MPEG25] = samplesPerFrame[MPEG2]
}

// NewDecoder returns a decoder that will process the provided reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r, nil}
}

// fill slice d until it is of len l, using bytes from reader r
func fillbuf(d []byte, r io.Reader, l int) (res []byte, err error) {
	if len(d) >= l {
		// we already have enough bytes
		return d, nil
	}

	// How many bytes do we need to fetch
	missing := l - len(d)

	// Does d have sufficient capacity? if not extent it
	if cap(d) < l {
		if d == nil {
			d = make([]byte, l)
		} else {
			il := len(d)
			d = d[:cap(d)] // stretch d to it's full capacity
			d = append(d, make([]byte, l-cap(d))...)
			d = d[:il] //we've extended the capm reset len
		}
	}

	d = d[:l]
	_, err = io.ReadFull(r, d[len(d)-missing:])

	return d, err
}

// Decode reads the next complete discovered frame into the provided
// Frame struct. A count of skipped bytes will be written to skipped.
func (d *Decoder) Decode(v *Frame, skipped *int) (err error) {
	// Truncate the array
	v.buf = v.buf[:0]

	hLen := 4
	// locate a sync frame
	*skipped = 0
	for {
		v.buf, err = fillbuf(v.buf, d.src, hLen)
		if err != nil {
			return err
		}
		if v.buf[0] == 0xFF && (v.buf[1]&0xE0 == 0xE0) &&
			v.Header().Emphasis() != EmphReserved &&
			v.Header().Layer() != LayerReserved &&
			v.Header().Version() != MPEGReserved &&
			v.Header().SampleRate() != -1 &&
			v.Header().BitRate() != -1 {
			break
		}
		switch {
		case v.buf[1] == 0xFF:
			v.buf = v.buf[1:]
			*skipped++
		default:
			v.buf = v.buf[2:]
			*skipped += 2
		}
	}

	crcLen := 0
	if v.Header().Protection() {
		crcLen = 2
		v.buf, err = fillbuf(v.buf, d.src, hLen+crcLen)
		if err != nil {
			return err
		}
	}

	sideLen, err := v.SideInfoLength()
	if err != nil {
		return err
	}

	v.buf, err = fillbuf(v.buf, d.src, hLen+crcLen+sideLen)
	if err != nil {
		return err
	}

	dataLen := v.Size()
	v.buf, err = fillbuf(v.buf, d.src, dataLen)
	if err != nil {
		return err
	}

	return nil
}

// SideInfoLength retursn the expected side info length for this
// mp3 frame
func (f *Frame) SideInfoLength() (int, error) {
	switch f.Header().Version() {
	case MPEG1:
		switch f.Header().ChannelMode() {
		case SingleChannel:
			return 17, nil
		case Stereo, JointStereo, DualChannel:
			return 32, nil
		default:
			return 0, errors.New("bad channel mode")
		}
	case MPEG2, MPEG25:
		switch f.Header().ChannelMode() {
		case SingleChannel:
			return 9, nil
		case Stereo, JointStereo, DualChannel:
			return 17, nil
		default:
			return 0, errors.New("bad channel mode")
		}
	default:
		return 0, fmt.Errorf("bad version (%v)", f.Header().Version())
	}
}

// Header returns the header for this frame
func (f *Frame) Header() FrameHeader {
	return FrameHeader(f.buf[0:4])
}

// CRC returns the CRC word stored in this frame
func (f *Frame) CRC() (uint16, error) {
	var crc uint16
	if !f.Header().Protection() {
		return 0, nil
	}
	crcdata := bytes.NewReader(f.buf[4:6])
	err := binary.Read(crcdata, binary.BigEndian, &crc)
	return crc, err
}

// SideInfo returns the  side info for this frame
func (f *Frame) SideInfo() FrameSideInfo {
	if f.Header().Protection() {
		return FrameSideInfo(f.buf[6:])
	}
	return FrameSideInfo(f.buf[4:])
}

// Frame returns a string describing this frame, header and side info
func (f *Frame) String() string {
	str := ""
	str += fmt.Sprintf("Header: \n%s", f.Header())
	str += fmt.Sprintf("SideInfo: \n%s", f.SideInfo())
	crc, err := f.CRC()
	str += fmt.Sprintf("CRC: %x (err: %v)\n", crc, err)
	str += fmt.Sprintf("Samples: %v\n", f.Samples())
	str += fmt.Sprintf("Size: %v\n", f.Size())
	str += fmt.Sprintf("Duration: %v\n", f.Duration())
	return str
}

// Version returns the MPEG version from the header
func (h FrameHeader) Version() FrameVersion {
	return FrameVersion((h[1] >> 3) & 0x03)
}

// Layer returns the MPEG layer from the header
func (h FrameHeader) Layer() FrameLayer {
	return FrameLayer((h[1] >> 1) & 0x03)
}

// Protection indicates if there is a CRC present after the header (before the side data)
func (h FrameHeader) Protection() bool {
	return (h[1] & 0x01) != 0x01
}

// BitRate returns the calculated bit rate from the header
func (h FrameHeader) BitRate() FrameBitRate {
	bitrateIdx := (h[2] >> 4) & 0x0F
	if bitrateIdx == 0x0F {
		return ErrInvalidBitrate
	}
	br := bitrates[h.Version()][h.Layer()][bitrateIdx] * 1000
	if br == 0 {
		return ErrInvalidBitrate
	}
	return FrameBitRate(br)
}

// SampleRate returns the samplerate from the header
func (h FrameHeader) SampleRate() FrameSampleRate {
	sri := (h[2] >> 2) & 0x03
	if sri == 0x03 {
		return ErrInvalidSampleRate
	}
	return FrameSampleRate(sampleRates[h.Version()][sri])
}

// Pad returns the pad bit, indicating if there are extra samples
// in this frame to make up the correct bitrate
func (h FrameHeader) Pad() bool {
	return ((h[2] >> 1) & 0x01) == 0x01
}

// Private retrusn the Private bit from the header
func (h FrameHeader) Private() bool {
	return (h[2] & 0x01) == 0x01
}

// ChannelMode returns the channel mode from the header
func (h FrameHeader) ChannelMode() FrameChannelMode {
	return FrameChannelMode((h[3] >> 6) & 0x03)
}

// CopyRight returns the CopyRight bit from the header
func (h FrameHeader) CopyRight() bool {
	return (h[3]>>3)&0x01 == 0x01
}

// Original returns the "original content" bit from the header
func (h FrameHeader) Original() bool {
	return (h[3]>>2)&0x01 == 0x01
}

// Emphasis returns the Emphasis from the header
func (h FrameHeader) Emphasis() FrameEmphasis {
	return FrameEmphasis((h[3] & 0x03))
}

// String dumps the frame header as a string for display purposes
func (h FrameHeader) String() string {
	str := ""
	str += fmt.Sprintf(" Layer: %v\n", h.Layer())
	str += fmt.Sprintf(" Version: %v\n", h.Version())
	str += fmt.Sprintf(" Protection: %v\n", h.Protection())
	str += fmt.Sprintf(" BitRate: %v\n", h.BitRate())
	str += fmt.Sprintf(" SampleRate: %v\n", h.SampleRate())
	str += fmt.Sprintf(" Pad: %v\n", h.Pad())
	str += fmt.Sprintf(" Private: %v\n", h.Private())
	str += fmt.Sprintf(" ChannelMode: %v\n", h.ChannelMode())
	str += fmt.Sprintf(" CopyRight: %v\n", h.CopyRight())
	str += fmt.Sprintf(" Original: %v\n", h.Original())
	str += fmt.Sprintf(" Emphasis: %v\n", h.Emphasis())
	return str
}

// NDataBegin is the number of bytes before the frame header at which the sample data begins
// 0 indicates that the data begins after the side channel information. This data is the
// data from the "bit reservoir" and can be up to 511 bytes
func (i FrameSideInfo) NDataBegin() uint16 {
	return (uint16(i[0]) << 1 & (uint16(i[1]) >> 7))
}

// Samples determines the number of samples based on the MPEG version and Layer from the header
func (f *Frame) Samples() int {
	return samplesPerFrame[f.Header().Version()][f.Header().Layer()]
}

// Size clculates the expected size of this frame in bytes based on the header
// information
func (f *Frame) Size() int {
	bps := float64(f.Samples()) / 8
	fsize := (bps * float64(f.Header().BitRate())) / float64(f.Header().SampleRate())
	if f.Header().Pad() {
		fsize += float64(slotSize[f.Header().Layer()])
	}
	return int(fsize)
}

// Duration calculates the time duration of this frame based on the samplerate and number of samples
func (f *Frame) Duration() time.Duration {
	ms := (1000 / float64(f.Header().SampleRate())) * float64(f.Samples())
	return time.Duration(int(float64(time.Millisecond) * ms))
}

// String renders the side info as a string for display purposes
func (i FrameSideInfo) String() string {
	str := ""
	str += fmt.Sprintf(" NDataBegin: %v\n", i.NDataBegin())
	return str
}

// Reader returns an io.Reader that reads the individual bytes from the frame
func (f *Frame) Reader() io.Reader {
	return bytes.NewReader(f.buf)
}
