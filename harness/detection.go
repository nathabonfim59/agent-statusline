package harness

type Detector func([]byte) (Harness, error)

var detectors []Detector

func Register(d Detector) {
	detectors = append(detectors, d)
}

func Detect(raw []byte) Harness {
	for _, d := range detectors {
		if h, err := d(raw); err == nil {
			return h
		}
	}
	return nil
}
