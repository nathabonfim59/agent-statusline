package harness

type Detector func([]byte) (Harness, error)

type Constructor func() Harness

var (
	detectors    []Detector
	constructors = map[string]Constructor{}
)

func Register(d Detector) {
	detectors = append(detectors, d)
}

func RegisterNamed(name string, c Constructor) {
	constructors[name] = c
}

func Detect(raw []byte) Harness {
	for _, d := range detectors {
		if h, err := d(raw); err == nil {
			return h
		}
	}
	return nil
}

func NewHarness(name string) Harness {
	if c, ok := constructors[name]; ok {
		return c()
	}
	return nil
}
