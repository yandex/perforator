package solomon

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"time"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

const HeaderSize = 24

type spackVersion uint16

const (
	version11 spackVersion = 0x0101
	version12 spackVersion = 0x0102
	version13 spackVersion = 0x0103
	version14 spackVersion = 0x0104
)

type spackFlag byte

const (
	memOnlyFlag   spackFlag = 0b0000_0001
	startTimeFlag spackFlag = 0b0000_0010
)

var packageInitTimeSeconds = uint32(time.Now().Unix())

func writeUint8(w io.Writer, v uint8) error {
	if bw, ok := w.(io.ByteWriter); ok {
		return bw.WriteByte(v)
	}
	_, err := w.Write([]byte{v})
	return err
}

func writeUint32LE(w io.Writer, v uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeUint64LE(w io.Writer, v uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeFloat64LE(w io.Writer, v float64) error {
	return writeUint64LE(w, math.Float64bits(v))
}

func writeULEB128(w io.Writer, value uint32) error {
	if bw, ok := w.(io.ByteWriter); ok {
		for value >= 0x80 {
			if err := bw.WriteByte(byte(value) | 0x80); err != nil {
				return err
			}
			value >>= 7
		}
		return bw.WriteByte(byte(value))
	}
	var buf [5]byte
	i := 0
	for value >= 0x80 {
		buf[i] = byte(value) | 0x80
		i++
		value >>= 7
	}
	buf[i] = byte(value)
	_, err := w.Write(buf[:i+1])
	return err
}

type spackMetric struct {
	flags     uint8
	startTime uint32

	nameValueIndex uint32
	labelsCount    uint32
	labelsStart    uint32
	labelsEnd      uint32

	metric Metric
}

func (se *spackEncoder) addValue(value string) (uint32, error) {
	if idx, ok := se.valuesIdx[value]; ok {
		return idx, nil
	}
	idx := se.valueCounter
	se.valuesIdx[value] = se.valueCounter
	se.valueCounter++

	if _, err := se.labelValuePool.WriteString(value); err != nil {
		return 0, err
	}
	if err := se.labelValuePool.WriteByte(0); err != nil {
		return 0, err
	}

	return idx, nil
}

func (se *spackEncoder) addName(name string) (uint32, error) {
	if idx, ok := se.namesIdx[name]; ok {
		return idx, nil
	}
	idx := se.nameCounter
	se.namesIdx[name] = idx
	se.nameCounter++

	if _, err := se.labelNamePool.WriteString(name); err != nil {
		return 0, err
	}
	if err := se.labelNamePool.WriteByte(0); err != nil {
		return 0, err
	}
	return idx, nil
}

func (se *spackEncoder) addValueLengthPrefixed(value string) (uint32, error) {
	if idx, ok := se.valuesIdx[value]; ok {
		return idx, nil
	}
	idx := se.valueCounter
	se.valuesIdx[value] = idx
	se.valueCounter++

	if err := writePoolStringLengthPrefixed(&se.labelValuePool, value); err != nil {
		return 0, err
	}
	return idx, nil
}

func (se *spackEncoder) addNameLengthPrefixed(name string) (uint32, error) {
	if idx, ok := se.namesIdx[name]; ok {
		return idx, nil
	}
	idx := se.nameCounter
	se.namesIdx[name] = idx
	se.nameCounter++

	if err := writePoolStringLengthPrefixed(&se.labelNamePool, name); err != nil {
		return 0, err
	}
	return idx, nil
}

func writePoolStringLengthPrefixed(pool *bytes.Buffer, value string) error {
	if uint64(len(value)) > math.MaxUint32 {
		return xerrors.New("SPACK string length exceeds uint32")
	}
	if err := writeULEB128(pool, uint32(len(value))); err != nil {
		return err
	}
	_, err := pool.WriteString(value)
	return err
}

func (s *spackMetric) writeLabel(se *spackEncoder, name string, value string) error {
	s.labelsCount++

	nameIdx, err := se.addName(name)
	if err != nil {
		return err
	}
	valueIdx, err := se.addValue(value)
	if err != nil {
		return err
	}

	if err := writeULEB128(&se.labelsBuf, nameIdx); err != nil {
		return err
	}
	return writeULEB128(&se.labelsBuf, valueIdx)
}

func (s *spackMetric) writeLabelLengthPrefixed(se *spackEncoder, name string, value string) error {
	s.labelsCount++

	nameIdx, err := se.addNameLengthPrefixed(name)
	if err != nil {
		return err
	}
	valueIdx, err := se.addValueLengthPrefixed(value)
	if err != nil {
		return err
	}

	if err := writeULEB128(&se.labelsBuf, nameIdx); err != nil {
		return err
	}
	return writeULEB128(&se.labelsBuf, valueIdx)
}

func (s *spackMetric) writeMetric(w io.Writer, version spackVersion, labelsBuf []byte) error {
	_, isSeries := s.metric.(seriesMetric)

	metricValueType := valueTypeOneWithoutTS
	ts := s.metric.getTimestamp()
	switch {
	case isSeries:
		metricValueType = valueTypeManyWithTS
	case ts != nil:
		metricValueType = valueTypeOneWithTS
	}

	mType := s.metric.getType()
	types := uint8(mType<<2) | uint8(metricValueType)

	var hdr [2]byte
	hdr[0] = types
	hdr[1] = s.flags
	if _, err := w.Write(hdr[:]); err != nil {
		return xerrors.Errorf("write types and flags failed: %w", err)
	}
	if s.flags&uint8(startTimeFlag) != 0 {
		if err := writeUint32LE(w, s.startTime); err != nil {
			return xerrors.Errorf("write start time failed: %w", err)
		}
	}

	if version == version12 {
		if err := writeULEB128(w, s.nameValueIndex); err != nil {
			return xerrors.Errorf("write name value index failed: %w", err)
		}
	}

	if err := writeULEB128(w, s.labelsCount); err != nil {
		return xerrors.Errorf("write labels count failed: %w", err)
	}

	if _, err := w.Write(labelsBuf[s.labelsStart:s.labelsEnd]); err != nil {
		return xerrors.Errorf("write labels failed: %w", err)
	}
	if !isSeries && ts != nil {
		if err := writeUint32LE(w, uint32(ts.Unix())); err != nil {
			return xerrors.Errorf("write timestamp failed: %w", err)
		}
	}

	if err := s.metric.writeSpackValue(w); err != nil {
		return xerrors.Errorf("write metric value failed: %w", err)
	}
	return nil
}

func (s *spackMetric) calculateMetricFlags() uint8 {
	var flags uint8
	if s.metric.isMemOnly() {
		flags |= uint8(memOnlyFlag)
	}
	return flags
}

func (s *spackMetric) addStartTimeFlag(commonStartTime uint32) {
	mType := s.metric.getType()
	startTime := s.metric.getStartTime()
	if startTime == 0 || (mType != typeRated && mType != typeRatedHistogram) {
		return
	}
	s.startTime = startTime
	if startTime != commonStartTime && startTime != 0 {
		s.flags |= uint8(startTimeFlag)
	}
}

func (s *spackMetric) getMetricNameValue(name string) string {
	value := s.metric.Name()

	if lvalue, ok := s.metric.Labels()[name]; ok {
		value = lvalue
	}

	return value
}

type SpackOpts func(*spackEncoder)

func WithVersion12() func(*spackEncoder) {
	return func(se *spackEncoder) {
		se.version = version12
	}
}

// WithVersion13 enables length-delimited string pools, allowing NUL bytes in labels.
func WithVersion13() func(*spackEncoder) {
	return func(se *spackEncoder) {
		se.version = version13
	}
}

// WithVersion14 enables rate start times in addition to the SPACK 1.3 format.
func WithVersion14() func(*spackEncoder) {
	return func(se *spackEncoder) {
		se.version = version14
	}
}

type labelIdxPair struct {
	nameIdx  uint32
	valueIdx uint32
}

type spackEncoder struct {
	context         context.Context
	compression     uint8
	version         spackVersion
	commonStartTime uint32

	nameCounter  uint32
	valueCounter uint32

	labelNamePool  bytes.Buffer
	labelValuePool bytes.Buffer
	labelsBuf      bytes.Buffer

	namesIdx  map[string]uint32
	valuesIdx map[string]uint32

	commonLabelIdx []labelIdxPair

	metrics Metrics
}

func NewSpackEncoder(ctx context.Context, compression CompressionType, metrics *Metrics, opts ...SpackOpts) *spackEncoder {
	if metrics == nil {
		metrics = &Metrics{}
	}

	namesHint := 16
	valuesHint := len(metrics.metrics) + namesHint

	se := &spackEncoder{
		context:         ctx,
		compression:     uint8(compression),
		version:         version11,
		commonStartTime: packageInitTimeSeconds,
		metrics:         *metrics,
		namesIdx:        make(map[string]uint32, namesHint),
		valuesIdx:       make(map[string]uint32, valuesHint),
	}
	if metrics.commonStartTime != 0 {
		se.commonStartTime = metrics.commonStartTime
	}
	if n := len(metrics.metrics); n > 0 {
		se.labelsBuf.Grow(n * namesHint)
	}
	for _, op := range opts {
		op(se)
	}
	return se
}

func (se *spackEncoder) writeLabels() ([]spackMetric, error) {
	spackMetrics := make([]spackMetric, len(se.metrics.metrics))
	// Dispatch outside the loops so legacy string pools do not check for 1.3.
	switch se.version {
	case version11:
		if err := se.processCommonLabels(); err != nil {
			return nil, err
		}
		for idx, metric := range se.metrics.metrics {
			if err := se.processMetricV11(&spackMetrics[idx], metric); err != nil {
				return nil, err
			}
		}
	case version12:
		if err := se.processCommonLabels(); err != nil {
			return nil, err
		}
		for idx, metric := range se.metrics.metrics {
			if err := se.processMetricV12(&spackMetrics[idx], metric); err != nil {
				return nil, err
			}
		}
	case version13:
		if err := se.processCommonLabelsLengthPrefixed(); err != nil {
			return nil, err
		}
		for idx, metric := range se.metrics.metrics {
			if err := se.processMetricV13(&spackMetrics[idx], metric); err != nil {
				return nil, err
			}
		}
	case version14:
		if err := se.processCommonLabelsLengthPrefixed(); err != nil {
			return nil, err
		}
		for idx, metric := range se.metrics.metrics {
			if err := se.processMetricV14(&spackMetrics[idx], metric); err != nil {
				return nil, err
			}
		}
	default:
		return nil, xerrors.Errorf("unsupported version: %v", se.version)
	}

	return spackMetrics, nil
}

func (se *spackEncoder) processCommonLabels() error {
	commonLabels := se.metrics.CommonLabels()
	if len(commonLabels) == 0 {
		return nil
	}
	se.commonLabelIdx = make([]labelIdxPair, 0, len(commonLabels))
	for name, value := range commonLabels {
		nameIdx, err := se.addName(name)
		if err != nil {
			return err
		}
		valueIdx, err := se.addValue(value)
		if err != nil {
			return err
		}
		se.commonLabelIdx = append(se.commonLabelIdx, labelIdxPair{nameIdx: nameIdx, valueIdx: valueIdx})
	}
	return nil
}

func (se *spackEncoder) processCommonLabelsLengthPrefixed() error {
	commonLabels := se.metrics.CommonLabels()
	if len(commonLabels) == 0 {
		return nil
	}
	se.commonLabelIdx = make([]labelIdxPair, 0, len(commonLabels))
	for name, value := range commonLabels {
		nameIdx, err := se.addNameLengthPrefixed(name)
		if err != nil {
			return err
		}
		valueIdx, err := se.addValueLengthPrefixed(value)
		if err != nil {
			return err
		}
		se.commonLabelIdx = append(se.commonLabelIdx, labelIdxPair{nameIdx: nameIdx, valueIdx: valueIdx})
	}
	return nil
}

func (se *spackEncoder) processMetricV11(m *spackMetric, metric Metric) error {
	m.metric = metric
	m.flags = m.calculateMetricFlags()
	m.labelsStart = uint32(se.labelsBuf.Len())

	nameTag := metric.getNameTag()
	if err := m.writeLabel(se, nameTag, m.getMetricNameValue(nameTag)); err != nil {
		return err
	}

	labels := metric.Labels()
	commonLabels := se.metrics.CommonLabels()

	for name, value := range labels {
		if name == nameTag {
			continue
		}
		if cValue, ok := commonLabels[name]; ok && cValue == value {
			continue
		}
		if err := m.writeLabel(se, name, value); err != nil {
			return err
		}
	}

	m.labelsEnd = uint32(se.labelsBuf.Len())
	return nil
}

func (se *spackEncoder) processMetricV12(m *spackMetric, metric Metric) error {
	m.metric = metric
	m.flags = m.calculateMetricFlags()
	m.labelsStart = uint32(se.labelsBuf.Len())

	idx, err := se.addValue(m.getMetricNameValue("name"))
	if err != nil {
		return err
	}
	m.nameValueIndex = idx

	labels := metric.Labels()
	nameTag := metric.getNameTag()
	commonLabels := se.metrics.CommonLabels()

	for name, value := range labels {
		if name == nameTag || name == "name" {
			continue
		}
		if cValue, ok := commonLabels[name]; ok && cValue == value {
			continue
		}
		if err := m.writeLabel(se, name, value); err != nil {
			return err
		}
	}

	m.labelsEnd = uint32(se.labelsBuf.Len())
	return nil
}

func (se *spackEncoder) processMetricV13(m *spackMetric, metric Metric) error {
	m.metric = metric
	m.flags = m.calculateMetricFlags()
	m.labelsStart = uint32(se.labelsBuf.Len())

	nameTag := metric.getNameTag()
	if err := m.writeLabelLengthPrefixed(se, nameTag, m.getMetricNameValue(nameTag)); err != nil {
		return err
	}

	labels := metric.Labels()
	commonLabels := se.metrics.CommonLabels()

	for name, value := range labels {
		if name == nameTag {
			continue
		}
		if cValue, ok := commonLabels[name]; ok && cValue == value {
			continue
		}
		if err := m.writeLabelLengthPrefixed(se, name, value); err != nil {
			return err
		}
	}

	m.labelsEnd = uint32(se.labelsBuf.Len())
	return nil
}

func (se *spackEncoder) processMetricV14(m *spackMetric, metric Metric) error {
	if err := se.processMetricV13(m, metric); err != nil {
		return err
	}
	m.addStartTimeFlag(se.commonStartTime)
	return nil
}

func (se *spackEncoder) Encode(w io.Writer) (written int, err error) {
	if cerr := se.context.Err(); cerr != nil {
		return 0, xerrors.Errorf("streamSpack context error: %w", cerr)
	}

	spackMetrics, err := se.writeLabels()
	if err != nil {
		return written, xerrors.Errorf("writeLabels failed: %w", err)
	}

	err = se.writeHeader(w)
	if err != nil {
		return written, xerrors.Errorf("writeHeader failed: %w", err)
	}
	written += HeaderSize
	compression := CompressionType(se.compression)

	cw := newCompressedWriter(w, compression)
	if lz4cw, ok := cw.(*lz4CompressionWriteCloser); ok {
		defer releaseLZ4Writer(lz4cw)
	}

	err = se.writeLabelNamesPool(cw)
	if err != nil {
		return written, xerrors.Errorf("writeLabelNamesPool failed: %w", err)
	}

	err = se.writeLabelValuesPool(cw)
	if err != nil {
		return written, xerrors.Errorf("writeLabelValuesPool failed: %w", err)
	}

	err = se.writeCommonTime(cw)
	if err != nil {
		return written, xerrors.Errorf("writeCommonTime failed: %w", err)
	}
	if se.version == version14 {
		if err = writeUint32LE(cw, se.commonStartTime); err != nil {
			return written, xerrors.Errorf("writeCommonStartTime failed: %w", err)
		}
	}

	err = se.writeCommonLabels(cw)
	if err != nil {
		return written, xerrors.Errorf("writeCommonLabels failed: %w", err)
	}

	err = se.writeMetricsData(cw, spackMetrics)
	if err != nil {
		return written, xerrors.Errorf("writeMetricsData failed: %w", err)
	}

	err = cw.Close()
	if err != nil {
		return written, xerrors.Errorf("close failed: %w", err)
	}

	switch compression {
	case CompressionNone:
		written += cw.(*noCompressionWriteCloser).written
	case CompressionLz4:
		written += cw.(*lz4CompressionWriteCloser).written
	}

	return written, nil
}

func (se *spackEncoder) writeHeader(w io.Writer) error {
	var totalPoints uint32
	for _, m := range se.metrics.metrics {
		if sm, ok := m.(seriesMetric); ok {
			totalPoints += sm.pointsCount()
		} else {
			totalPoints++
		}
	}

	namesSize := uint32(se.labelNamePool.Len())
	valuesSize := uint32(se.labelValuePool.Len())
	if se.version == version13 || se.version == version14 {
		// Since 1.3 these header fields count strings, not bytes.
		namesSize = se.nameCounter
		valuesSize = se.valueCounter
	}

	var buf [HeaderSize]byte
	binary.LittleEndian.PutUint16(buf[0:2], 0x5053)                            // Magic.
	binary.LittleEndian.PutUint16(buf[2:4], uint16(se.version))                // Version.
	binary.LittleEndian.PutUint16(buf[4:6], HeaderSize)                        // Header size.
	buf[6] = 0                                                                 // TimePrecision = SECONDS
	buf[7] = se.compression                                                    // CompressionAlg
	binary.LittleEndian.PutUint32(buf[8:12], namesSize)                        // Label names size.
	binary.LittleEndian.PutUint32(buf[12:16], valuesSize)                      // Label values size.
	binary.LittleEndian.PutUint32(buf[16:20], uint32(len(se.metrics.metrics))) // Metrics count.
	binary.LittleEndian.PutUint32(buf[20:24], totalPoints)                     // Points count.

	if _, err := w.Write(buf[:]); err != nil {
		return xerrors.Errorf("write header failed: %w", err)
	}
	return nil
}

func (se *spackEncoder) writeLabelNamesPool(w io.Writer) error {
	_, err := w.Write(se.labelNamePool.Bytes())
	if err != nil {
		return xerrors.Errorf("write labelNamePool failed: %w", err)
	}
	return nil
}

func (se *spackEncoder) writeLabelValuesPool(w io.Writer) error {
	_, err := w.Write(se.labelValuePool.Bytes())
	if err != nil {
		return xerrors.Errorf("write labelValuePool failed: %w", err)
	}
	return nil
}

func (se *spackEncoder) writeCommonTime(w io.Writer) error {
	var ts uint32
	if se.metrics.timestamp != nil {
		ts = uint32(se.metrics.timestamp.Unix())
	}
	return writeUint32LE(w, ts)
}

func (se *spackEncoder) writeCommonLabels(w io.Writer) error {
	if len(se.commonLabelIdx) == 0 {
		return writeUint8(w, 0)
	}

	if err := writeULEB128(w, uint32(len(se.commonLabelIdx))); err != nil {
		return xerrors.Errorf("write commonLabels count failed: %w", err)
	}

	for _, pair := range se.commonLabelIdx {
		if err := writeULEB128(w, pair.nameIdx); err != nil {
			return xerrors.Errorf("write commonLabel name index failed: %w", err)
		}
		if err := writeULEB128(w, pair.valueIdx); err != nil {
			return xerrors.Errorf("write commonLabel value index failed: %w", err)
		}
	}
	return nil
}

func (se *spackEncoder) writeMetricsData(w io.Writer, metrics []spackMetric) error {
	labelsBuf := se.labelsBuf.Bytes()
	for i := range metrics {
		// Check context once per 63 iterations.
		// i&0x3f effectively equals to i%63 but compiles to 1 assembly instruction instead of 6.
		if i&0x3f == 0 {
			if err := se.context.Err(); err != nil {
				return xerrors.Errorf("streamSpack context error: %w", err)
			}
		}

		if err := metrics[i].writeMetric(w, se.version, labelsBuf); err != nil {
			return xerrors.Errorf("write metric failed: %w", err)
		}
	}
	return nil
}
