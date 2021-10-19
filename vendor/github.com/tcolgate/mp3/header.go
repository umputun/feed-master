package mp3

/*
func (this *FrameHeader) Parse(bs []byte) error {
	this.Size = 0
	this.Samples = 0
	this.Duration = 0

	if len(bs) < 4 {
		return fmt.Errorf("not enough bytes")
	}
	if bs[0] != 0xFF || (bs[1]&0xE0) != 0xE0 {
		return fmt.Errorf("missing sync word, got: %x, %x", bs[0], bs[1])
	}
	this.Version = Version((bs[1] >> 3) & 0x03)
	if this.Version == MPEGReserved {
		return fmt.Errorf("reserved mpeg version")
	}

	this.Layer = Layer(((bs[1] >> 1) & 0x03))
	if this.Layer == LayerReserved {
		return fmt.Errorf("reserved layer")
	}

	this.Protection = (bs[1] & 0x01) != 0x01

	bitrateIdx := (bs[2] >> 4) & 0x0F
	if bitrateIdx == 0x0F {
		return fmt.Errorf("invalid bitrate: %v", bitrateIdx)
	}
	this.Bitrate = bitrates[this.Version][this.Layer][bitrateIdx] * 1000
	if this.Bitrate == 0 {
		return fmt.Errorf("invalid bitrate: %v", bitrateIdx)
	}

	sampleRateIdx := (bs[2] >> 2) & 0x03
	if sampleRateIdx == 0x03 {
		return fmt.Errorf("invalid sample rate: %v", sampleRateIdx)
	}
	this.SampleRate = sampleRates[this.Version][sampleRateIdx]

	this.Pad = ((bs[2] >> 1) & 0x01) == 0x01

	this.Private = (bs[2] & 0x01) == 0x01

	this.ChannelMode = ChannelMode(bs[3]>>6) & 0x03

	// todo: mode extension

	this.CopyRight = (bs[3]>>3)&0x01 == 0x01

	this.Original = (bs[3]>>2)&0x01 == 0x01

	this.Emphasis = Emphasis(bs[3] & 0x03)
	if this.Emphasis == EmphReserved {
		return fmt.Errorf("reserved emphasis")
	}

	this.Size = this.size()
	this.Samples = this.samples()
	this.Duration = this.duration()

	return nil
}

func (this *FrameHeader) samples() int {
	return samplesPerFrame[this.Version][this.Layer]
}

func (this *FrameHeader) size() int64 {
	bps := float64(this.samples()) / 8
	fsize := (bps * float64(this.Bitrate)) / float64(this.SampleRate)
	if this.Pad {
		fsize += float64(slotSize[this.Layer])
	}
	return int64(fsize)
}

func (this *FrameHeader) duration() time.Duration {
	ms := (1000 / float64(this.SampleRate)) * float64(this.samples())
	return time.Duration(time.Duration(float64(time.Millisecond) * ms))
}
*/
