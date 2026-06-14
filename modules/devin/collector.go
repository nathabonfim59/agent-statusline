package devin

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
)

type DevinData struct {
	Model        string     `json:"model"`
	InputTokens  int        `json:"input_tokens"`
	OutputTokens int        `json:"output_tokens"`
	Quota        *QuotaInfo `json:"quota,omitempty"`
}

type Collector struct {
	mu    sync.RWMutex
	data  DevinData
	debug bool
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) SetDebug(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.debug = on
}

func (c *Collector) HandleResponse(host, path, contentType string, body []byte) {
	if len(body) == 0 {
		return
	}

	if contains(path, "GetChatMessage") {
		c.handleChatMessage(body)
	} else if contains(path, "GetUserStatus") {
		c.handleUserStatus(body)
	} else if contains(path, "/v3/self") {
		c.handleSelf(body)
	}
}

func (c *Collector) GetData() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

func (c *Collector) handleChatMessage(data []byte) {
	msgs := parseEnvelopes(data)
	if len(msgs) < 3 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.debug {
		dumpMessages(msgs)
	}

	// Only process complete responses — the last message is the [15] end marker.
	if !isEndMarker(msgs[len(msgs)-1]) {
		return
	}

	// Content message is 2 before end, stats trailer is 1 before end.
	contentMsg := msgs[len(msgs)-3]
	statsMsg := msgs[len(msgs)-2]

	model := extractModel(contentMsg)
	sit, sot := extractUsageStats(statsMsg)
	if sit == 0 && sot == 0 {
		return
	}

	// Compactor: update tokens only, keep current model.
	if model == "compactor" {
		c.data.InputTokens = sit + sot
		c.data.OutputTokens = sot
		return
	}

	// Subagent: content message carries top-level field 5 (StopReason).
	if hasTopLevelField(contentMsg, 5) {
		return
	}

	if model != "" {
		c.data.Model = model
	}
	c.data.InputTokens = sit + sot
	c.data.OutputTokens = sot
}

func (c *Collector) handleUserStatus(data []byte) {
	quota := extractQuota(data)
	if quota == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.Quota = quota
}

func (c *Collector) handleSelf(data []byte) {
	// /v3/self returns JSON - just store any relevant info
	var self struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if json.Unmarshal(data, &self) == nil {
		// Could use self info if needed
	}
}

func parseEnvelopes(data []byte) [][]byte {
	var msgs [][]byte
	pos := 0
	for pos+5 <= len(data) {
		flags := data[pos]
		length := binary.BigEndian.Uint32(data[pos+1 : pos+5])
		pos += 5
		if length > 10_000_000 {
			break
		}
		if pos+int(length) <= len(data) {
			msgs = append(msgs, data[pos:pos+int(length)])
			pos += int(length)
		} else {
			break
		}
		_ = flags
	}
	return msgs
}

func extractTokens(msg []byte) (inputTokens, outputTokens int) {
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(msg, pos)
			pos += lb
			if pos+length <= len(msg) {
				sub := msg[pos : pos+length]
				pos += length
				if fn == 7 {
					sp := 0
					for sp < len(sub) {
						st, svb := readVarint(sub, sp)
						if svb == 0 {
							break
						}
						sp += svb
						sf := st >> 3
						sw := st & 0x7
						if sw == 0 {
							val, vv := readVarint(sub, sp)
							sp += vv
							if sf == 2 {
								inputTokens = val
							} else if sf == 3 {
								outputTokens = val
							}
						} else if sw == 2 {
							slen, slb := readVarint(sub, sp)
							sp += slb + slen
						} else {
							sp += 4
							if sw == 1 {
								sp += 4
							}
						}
					}
				}
			} else {
				break
			}
		} else if wt == 0 {
			_, vv := readVarint(msg, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return
}

func extractUsageStats(msg []byte) (inputTokens, outputTokens int) {
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(msg, pos)
			pos += lb
			if pos+length > len(msg) {
				break
			}
			if fn == 28 {
				it, ot := parseStatsBlock(msg[pos : pos+length])
				if it > inputTokens {
					inputTokens = it
				}
				if ot > outputTokens {
					outputTokens = ot
				}
			}
			pos += length
		} else if wt == 0 {
			_, vv := readVarint(msg, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return
}

func parseStatsBlock(data []byte) (inputTokens, outputTokens int) {
	pos := 0
	for pos < len(data) {
		tag, vb := readVarint(data, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(data, pos)
			pos += lb
			if pos+length > len(data) {
				break
			}
			if fn == 2 {
				key, value := parseStatMetric(data[pos : pos+length])
				if key == "input_tokens" && value > inputTokens {
					inputTokens = value
				} else if key == "output_tokens" && value > outputTokens {
					outputTokens = value
				}
			}
			pos += length
		} else if wt == 0 {
			_, vv := readVarint(data, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return
}

func parseStatMetric(data []byte) (key string, value int) {
	pos := 0
	for pos < len(data) {
		tag, vb := readVarint(data, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(data, pos)
			pos += lb
			if pos+length > len(data) {
				break
			}
			if fn == 4 {
				value = parseStatValue(data[pos : pos+length])
			} else if fn == 5 {
				key = string(data[pos : pos+length])
			}
			pos += length
		} else if wt == 0 {
			_, vv := readVarint(data, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return
}

func parseStatValue(data []byte) int {
	pos := 0
	for pos < len(data) {
		tag, vb := readVarint(data, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 5 {
			if pos+4 > len(data) {
				break
			}
			if fn == 2 {
				return int(math.Round(float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))))
			}
			pos += 4
		} else if wt == 2 {
			length, lb := readVarint(data, pos)
			pos += lb + length
		} else if wt == 0 {
			_, vv := readVarint(data, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else {
			break
		}
	}
	return 0
}

func hasTopLevelField(msg []byte, targetField int) bool {
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if fn == targetField {
			return true
		}
		if wt == 0 {
			_, vv := readVarint(msg, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 2 {
			length, lb := readVarint(msg, pos)
			pos += lb + length
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return false
}

func isEndMarker(msg []byte) bool {
	fields := topLevelFields(msg)
	return len(fields) == 1 && fields[0] == 15
}

func dumpMessages(msgs [][]byte) {
	for i, msg := range msgs {
		sid := extractSessionID(msg)
		model := extractModel(msg)
		it, ot := extractTokens(msg)
		sit, sot := extractUsageStats(msg)
		hasStats := sit > 0 || sot > 0
		fields := dumpFields(msg)

		fmt.Fprintf(os.Stderr, "[devin] msg[%d] session=%s model=%s tokens(in=%d out=%d) stats(in=%d out=%d) hasStats=%v\n",
			i, sid, model, it, ot, sit, sot, hasStats)
		for _, f := range fields {
			fmt.Fprintf(os.Stderr, "  %s\n", f)
		}
	}
}

func dumpFields(msg []byte) []string {
	var out []string
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		switch wt {
		case 0:
			val, vv := readVarint(msg, pos)
			pos += vv
			out = append(out, fmt.Sprintf("field=%d varint=%d", fn, val))
		case 1:
			if pos+8 <= len(msg) {
				val := binary.LittleEndian.Uint64(msg[pos : pos+8])
				pos += 8
				out = append(out, fmt.Sprintf("field=%d fixed64=%d", fn, val))
			} else {
				pos = len(msg)
			}
		case 2:
			length, lb := readVarint(msg, pos)
			pos += lb
			if pos+length <= len(msg) {
				sub := msg[pos : pos+length]
				pos += length
				subFields := dumpFields(sub)
				if len(subFields) == 0 {
					out = append(out, fmt.Sprintf("field=%d bytes=%d raw=%q", fn, length, truncate(sub, 60)))
				} else {
					out = append(out, fmt.Sprintf("field=%d message(%d){", fn, length))
					for _, sf := range subFields {
						out = append(out, "  "+sf)
					}
					out = append(out, "}")
				}
			} else {
				pos = len(msg)
			}
		case 5:
			if pos+4 <= len(msg) {
				val := math.Float32frombits(binary.LittleEndian.Uint32(msg[pos : pos+4]))
				pos += 4
				out = append(out, fmt.Sprintf("field=%d fixed32=%f", fn, val))
			} else {
				pos = len(msg)
			}
		default:
			pos = len(msg)
		}
	}
	return out
}

func truncate(b []byte, max int) string {
	s := string(b)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func topLevelFields(msg []byte) []int {
	var fields []int
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		fields = append(fields, fn)
		if wt == 0 {
			_, vv := readVarint(msg, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 2 {
			length, lb := readVarint(msg, pos)
			pos += lb + length
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return fields
}

func extractSessionID(msg []byte) string {
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(msg, pos)
			pos += lb
			if fn == 1 && pos+length <= len(msg) {
				return string(msg[pos : pos+length])
			}
			pos += length
		} else if wt == 0 {
			_, vv := readVarint(msg, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return ""
}

func extractModel(msg []byte) string {
	pos := 0
	for pos < len(msg) {
		tag, vb := readVarint(msg, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(msg, pos)
			pos += lb
			if fn == 7 && pos+length <= len(msg) {
				sub := msg[pos : pos+length]
				sp := 0
				for sp < len(sub) {
					st, svb := readVarint(sub, sp)
					if svb == 0 {
						break
					}
					sp += svb
					sfn := st >> 3
					swt := st & 0x7
					if swt == 2 {
						slen, slb := readVarint(sub, sp)
						sp += slb
						if sfn == 9 {
							return string(sub[sp : sp+slen])
						}
						sp += slen
					} else if swt == 0 {
						_, vv := readVarint(sub, sp)
						sp += vv
					} else {
						sp += 4
						if swt == 1 {
							sp += 4
						}
					}
				}
			}
			pos += length
		} else if wt == 0 {
			_, vv := readVarint(msg, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return ""
}

func extractQuota(data []byte) *QuotaInfo {
	q := &QuotaInfo{}
	pos := 0
	for pos < len(data) {
		tag, vb := readVarint(data, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(data, pos)
			pos += lb
			if pos+length <= len(data) {
				sub := data[pos : pos+length]
				pos += length
				if fn == 2 {
					sp := 0
					for sp < len(sub) {
						st, svb := readVarint(sub, sp)
						if svb == 0 {
							break
						}
						sp += svb
						sfn := st >> 3
						swt := st & 0x7
						if swt == 0 {
							val, vv := readVarint(sub, sp)
							sp += vv
							if sfn == 7 {
								q.DailyLimit = val
							} else if sfn == 8 {
								q.DailyUsed = val
							}
						} else if swt == 2 {
							slen, slb := readVarint(sub, sp)
							sp += slb
							if sfn == 2 {
								q.Plan = string(sub[sp : sp+slen])
							}
							sp += slen
						} else {
							sp += 4
							if swt == 1 {
								sp += 4
							}
						}
					}
				}
			}
		} else if wt == 0 {
			_, vv := readVarint(data, pos)
			pos += vv
		} else {
			pos += 4
			if wt == 1 {
				pos += 4
			}
		}
	}
	if q.Plan == "" {
		return nil
	}
	return q
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
