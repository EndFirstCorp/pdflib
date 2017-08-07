package filter

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/pkg/errors"
)

var (
	errFlateMissingDecodeParmColumn    = errors.New("filter FlateDecode: missing decode parm: Columns")
	errFlateMissingDecodeParmPredictor = errors.New("filter FlateDecode: missing decode parm: Predictor")
	errFlatePostProcessing             = errors.New("filter FlateDecode: postprocessing failed")
)

type flate struct {
	baseFilter
}

// Encode implements encoding for a Flate filter.
func (f flate) Encode(r io.Reader) (*bytes.Buffer, error) {

	logDebugFilter.Println("EncodeFlate begin")

	// Optional decode parameters need preprocessing
	// but this filter implementation is used for object streams
	// and xref streams only and does not use decode parameters.

	var b bytes.Buffer
	w := zlib.NewWriter(&b)

	written, err := io.Copy(w, r)
	if err != nil {
		return nil, err
	}
	logDebugFilter.Printf("EncodeFlate: %d bytes written\n", written)

	w.Close()

	logDebugFilter.Println("EncodeFlate end")

	return &b, nil
}

// Decode implements decoding for a Flate filter.
func (f flate) Decode(r io.Reader) (*bytes.Buffer, error) {

	logDebugFilter.Println("DecodeFlate begin")

	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	logDebugFilter.Printf("DecodeFlate: decoded %d bytes.\n", written)

	rc.Close()

	if f.decodeParms == nil {
		logDebugFilter.Println("DecodeFlate end w/o decodeParms")
		return &b, nil
	}

	logDebugFilter.Println("DecodeFlate end w/o decodeParms")

	// Optional decode parameters need postprocessing.
	return f.decodePostProcess(&b)
}

// decodePostProcess
func (f flate) decodePostProcess(rin io.Reader) (a *bytes.Buffer, err error) {

	// The only postprocessing needed (for decoding object streams) is: PredictorUp with PngUp.

	const PredictorNo = 1
	const PredictorTIFF = 2
	const PredictorNone = 10
	const PredictorSub = 11
	const PredictorUp = 12 // implemented
	const PredictorAverage = 13
	const PredictorPaeth = 14
	const PredictorOptimum = 15

	const PngNone = 0x00
	const PngSub = 0x01
	const PngUp = 0x02 // implemented
	const PngAverage = 0x03
	const PngPaeth = 0x04

	c := f.decodeParms.IntEntry("Columns")
	if c == nil {
		err = errFlateMissingDecodeParmColumn
		return
	}

	columns := *c

	p := f.decodeParms.IntEntry("Predictor")
	if p == nil {
		err = errFlateMissingDecodeParmPredictor
		return
	}

	predictor := *p

	// PredictorUp is a popular predictor used for flate encoded stream dicts.
	if predictor != PredictorUp {
		err = errors.Errorf("Filter FlateDecode: Predictor %d unsupported", predictor)
		return
	}

	// BitsPerComponent optional, integer: 1,2,4,8,16 (Default:8)
	// The number of bits used to represent each colour component in a sample.
	bpc := f.decodeParms.IntEntry("BitsPerComponents")
	if bpc != nil {
		err = errors.Errorf("Filter FlateDecode: Unexpected \"BitsPerComponent\": %d", *bpc)
		return
	}

	// Colors, optional, integer: 1,2,3,4 (Default:1)
	// The number of interleaved colour components per sample.
	colors := f.decodeParms.IntEntry("Colors")
	if colors != nil {
		err = errors.Errorf("Filter FlateDecode: Unexpected \"Colors\": %d", *colors)
		return
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rin)
	if err != nil {
		return
	}

	b := buf.Bytes()

	if len(b)%(columns+1) > 0 {
		return nil, errFlatePostProcessing
	}

	var fbuf []byte
	j := 0
	for i := 0; i < len(b); i += columns + 1 {
		if b[i] != PngUp {
			err = errFlatePostProcessing
			return
		}
		fbuf = append(fbuf, b[i+1:i+columns+1]...)
		j++
	}

	bufOut := make([]byte, len(fbuf))
	for i := 0; i < len(fbuf); i += columns {
		for j := 0; j < columns; j++ {
			from := i - columns + j
			if from >= 0 {
				bufOut[i+j] = fbuf[i+j] + bufOut[i-columns+j]
			} else {
				bufOut[j] = fbuf[j]
			}
		}
	}

	return bytes.NewBuffer(bufOut), nil
}
