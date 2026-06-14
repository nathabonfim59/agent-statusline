package devin

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestExtractUsageStats(t *testing.T) {
	msg, err := hex.DecodeString("0a28626f742d35356662346638312d326461652d343833312d623861362d646336323233623333616531120c08f78bbcd10610a1d1a8de01e2016e0a13526573706f6e73652053746174697374696373123c2a0e6167656e745f6d65737361676573222a0a0e4167656e74206d65737361676573150000803f1a08206d6573736167652209206d6573736167657312192a056d6f64656c1a100a054d6f64656c12075357452d312e36e201ba010a0b546f6b656e20557361676512342a0c696e7075745f746f6b656e7322240a0c496e70757420746f6b656e731500587a461a0620746f6b656e220720746f6b656e7312362a0d6f75747075745f746f6b656e7322250a0d4f757470757420746f6b656e73150000ef431a0620746f6b656e220720746f6b656e73123d2a136361636865645f696e7075745f746f6b656e7322260a1343616368656420696e70757420746f6b656e731a0620746f6b656e220720746f6b656e73")
	if err != nil {
		t.Fatal(err)
	}

	input, output := extractUsageStats(msg)
	if input != 16022 || output != 478 {
		t.Fatalf("extractUsageStats() = %d, %d; want 16022, 478", input, output)
	}
}

func TestHandleChatMessageCompleteResponse(t *testing.T) {
	stats, err := hex.DecodeString("0a28626f742d35356662346638312d326461652d343833312d623861362d646336323233623333616531120c08f78bbcd10610a1d1a8de01e2016e0a13526573706f6e73652053746174697374696373123c2a0e6167656e745f6d65737361676573222a0a0e4167656e74206d65737361676573150000803f1a08206d6573736167652209206d6573736167657312192a056d6f64656c1a100a054d6f64656c12075357452d312e36e201ba010a0b546f6b656e20557361676512342a0c696e7075745f746f6b656e7322240a0c496e70757420746f6b656e731500587a461a0620746f6b656e220720746f6b656e7312362a0d6f75747075745f746f6b656e7322250a0d4f757470757420746f6b656e73150000ef431a0620746f6b656e220720746f6b656e73123d2a136361636865645f696e7075745f746f6b656e7322260a1343616368656420696e70757420746f6b656e731a0620746f6b656e220720746f6b656e73")
	if err != nil {
		t.Fatal(err)
	}

	// Content message: model=swe-1-6, input=16022, output=478 (field 7)
	contentMsg := []byte{
		0x3A, 0x0E, // field 7, length 14
		0x10, 0x96, 0xFA, 0x01, // field 2 varint 16022
		0x18, 0xDE, 0x03, // field 3 varint 478
		0x4A, 0x07, 's', 'w', 'e', '-', '1', '-', '6', // field 9 "swe-1-6"
	}
	// End marker: field 15
	endMarker := []byte{0x78, 0x00}

	c := NewCollector()

	// Main session: complete response with [15].
	c.handleChatMessage(connectStream(contentMsg, stats, endMarker))
	data := c.GetData().(DevinData)
	if data.Model != "swe-1-6" || data.InputTokens != 16500 || data.OutputTokens != 478 {
		t.Fatalf("main session: model=%s input=%d output=%d; want swe-1-6 16500 478",
			data.Model, data.InputTokens, data.OutputTokens)
	}

	// Subagent: smaller input, no compaction — should be ignored.
	subContent := []byte{
		0x3A, 0x0E,
		0x10, 0x9E, 0x01, // field 2 varint 158 (sum in+out = 49+109)
		0x18, 0x6D, // field 3 varint 109
		0x4A, 0x0D, 's', 'w', 'e', '-', '1', '-', '6', '-', 'f', 'a', 's', 't',
	}
	subStats := []byte{
		0xE2, 0x01, 0x2F, // field 28, length 47
		0x12, 0x15, // StatMetric for input_tokens, length 21
		0x22, 0x05, 0x15, 0x00, 0x00, 0x44, 0x42, // StatValue fixed32 49.0
		0x2A, 0x0C, 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x12, 0x16, // StatMetric for output_tokens, length 22
		0x22, 0x05, 0x15, 0x00, 0x00, 0xDA, 0x42, // StatValue fixed32 109.0
		0x2A, 0x0D, 'o', 'u', 't', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
	}
	c.handleChatMessage(connectStream(subContent, subStats, endMarker))
	data = c.GetData().(DevinData)
	if data.Model != "swe-1-6" || data.InputTokens != 16500 {
		t.Fatalf("after subagent: model=%s input=%d; want swe-1-6 16500",
			data.Model, data.InputTokens)
	}

	// Compaction: should update tokens, keep model, set justCompacted flag.
	compContent := []byte{
		0x3A, 0x0B,
		0x10, 0xA2, 0x28, // field 2 varint 5154 (4105+351+698 = sum)
		0x18, 0xDF, 0x02, // field 3 varint 351
		0x4A, 0x09, 'c', 'o', 'm', 'p', 'a', 'c', 't', 'o', 'r',
	}
	compStats := []byte{
		0xE2, 0x01, 0x2F,
		0x12, 0x15,
		0x22, 0x05, 0x15, 0x00, 0x48, 0x80, 0x45, // fixed32 4105.0
		0x2A, 0x0C, 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x12, 0x16,
		0x22, 0x05, 0x15, 0x00, 0x80, 0xAF, 0x43, // fixed32 351.0
		0x2A, 0x0D, 'o', 'u', 't', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
	}
	c.handleChatMessage(connectStream(compContent, compStats, endMarker))
	data = c.GetData().(DevinData)
	if data.Model != "swe-1-6" || data.InputTokens != 4456 {
		t.Fatalf("after compaction: model=%s input=%d; want swe-1-6 4456",
			data.Model, data.InputTokens)
	}

	// Post-compaction model switch: smaller input OK because justCompacted.
	postContent := []byte{
		0x3A, 0x0B,
		0x10, 0xA0, 0x14, // field 2 varint 2592 (2050+53+489)
		0x18, 0xD3, 0x03, // field 3 varint 467
		0x4A, 0x09, 'k', 'i', 'm', 'i', '-', 'k', '2', '-', '6',
	}
	postStats := []byte{
		0xE2, 0x01, 0x2F,
		0x12, 0x15,
		0x22, 0x05, 0x15, 0x00, 0x20, 0x00, 0x45, // fixed32 2050.0
		0x2A, 0x0C, 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x12, 0x16,
		0x22, 0x05, 0x15, 0x00, 0x80, 0xE9, 0x43, // fixed32 467.0
		0x2A, 0x0D, 'o', 'u', 't', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
	}
	c.handleChatMessage(connectStream(postContent, postStats, endMarker))
	data = c.GetData().(DevinData)
	if data.Model != "kimi-k2-6" || data.InputTokens != 2517 {
		t.Fatalf("post-compaction: model=%s input=%d; want kimi-k2-6 2517",
			data.Model, data.InputTokens)
	}
}

func connectStream(msgs ...[]byte) []byte {
	var out []byte
	for _, msg := range msgs {
		envelope := make([]byte, len(msg)+5)
		binary.BigEndian.PutUint32(envelope[1:5], uint32(len(msg)))
		copy(envelope[5:], msg)
		out = append(out, envelope...)
	}
	return out
}
