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

	input, output, cached := extractUsageStats(msg)
	if input != 16022 || output != 478 {
		t.Fatalf("extractUsageStats() = %d, %d; want 16022, 478", input, output)
	}
	if cached != 0 {
		t.Fatalf("extractUsageStats() cached = %d; want 0", cached)
	}
}

func TestHandleChatMessageCompleteResponse(t *testing.T) {
	oldLoadLocalData := loadLocalDataForCollector
	loadLocalDataForCollector = func() (*LocalData, ModelConfigs) {
		return &LocalData{Model: "swe-1-6"}, nil
	}
	defer func() { loadLocalDataForCollector = oldLoadLocalData }()

	stats, err := hex.DecodeString("0a28626f742d35356662346638312d326461652d343833312d623861362d646336323233623333616531120c08f78bbcd10610a1d1a8de01e2016e0a13526573706f6e73652053746174697374696373123c2a0e6167656e745f6d65737361676573222a0a0e4167656e74206d65737361676573150000803f1a08206d6573736167652209206d6573736167657312192a056d6f64656c1a100a054d6f64656c12075357452d312e36e201ba010a0b546f6b656e20557361676512342a0c696e7075745f746f6b656e7322240a0c496e70757420746f6b656e731500587a461a0620746f6b656e220720746f6b656e7312362a0d6f75747075745f746f6b656e7322250a0d4f757470757420746f6b656e73150000ef431a0620746f6b656e220720746f6b656e73123d2a136361636865645f696e7075745f746f6b656e7322260a1343616368656420696e70757420746f6b656e731a0620746f6b656e220720746f6b656e73")
	if err != nil {
		t.Fatal(err)
	}

	// Content message: model=swe-1-6, input=16022, output=478 (field 7)
	contentMsg := []byte{
		0x3A, 0x0C, // field 7, length 12
		0x10, 0x96, 0x7D, // field 2 varint 16022
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

	// Tiny same-model transient response should be ignored.
	subContent := []byte{
		0x28, 0x02, // field 5 varint=2 (StopReason)
		0x3A, 0x0D, // field 7, length 13
		0x10, 0x9E, 0x01, // field 2 varint 158
		0x18, 0x6D, // field 3 varint 109
		0x4A, 0x07, 's', 'w', 'e', '-', '1', '-', '6',
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

	otherModelContent := []byte{
		0x3A, 0x12,
		0x10, 0x31,
		0x18, 0x6D,
		0x4A, 0x0C, 's', 'w', 'e', '-', '1', '-', '6', '-', 'f', 'a', 's', 't',
	}
	c.handleChatMessage(connectStream(otherModelContent, subStats, endMarker))
	data = c.GetData().(DevinData)
	if data.Model != "swe-1-6" || data.InputTokens != 16500 {
		t.Fatalf("after different-model helper: model=%s input=%d; want swe-1-6 16500",
			data.Model, data.InputTokens)
	}

	// Compaction: should update tokens, keep model.
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
}

func TestLowerTokenModelSwitchRequiresLocalModel(t *testing.T) {
	oldLoadLocalData := loadLocalDataForCollector
	loadLocalDataForCollector = func() (*LocalData, ModelConfigs) {
		return &LocalData{Model: "swe-1-6-fast"}, nil
	}
	defer func() { loadLocalDataForCollector = oldLoadLocalData }()

	c := NewCollector()
	c.data = DevinData{Model: "swe-1-6", InputTokens: 16500, OutputTokens: 478}

	content := []byte{
		0x3A, 0x12,
		0x10, 0x31,
		0x18, 0x6D,
		0x4A, 0x0C, 's', 'w', 'e', '-', '1', '-', '6', '-', 'f', 'a', 's', 't',
	}
	stats := []byte{
		0xE2, 0x01, 0x2F,
		0x12, 0x15,
		0x22, 0x05, 0x15, 0x00, 0x00, 0x44, 0x42,
		0x2A, 0x0C, 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x12, 0x16,
		0x22, 0x05, 0x15, 0x00, 0x00, 0xDA, 0x42,
		0x2A, 0x0D, 'o', 'u', 't', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
	}
	endMarker := []byte{0x78, 0x00}

	c.handleChatMessage(connectStream(content, stats, endMarker))
	data := c.GetData().(DevinData)
	if data.Model != "swe-1-6-fast" || data.InputTokens != 158 {
		t.Fatalf("after selected model switch: model=%s input=%d; want swe-1-6-fast 158",
			data.Model, data.InputTokens)
	}
}

func TestLocalSessionChangeResetsTokenState(t *testing.T) {
	oldLoadLocalData := loadLocalDataForCollector
	session := "old"
	loadLocalDataForCollector = func() (*LocalData, ModelConfigs) {
		return &LocalData{SessionID: session, Model: "swe-1-6"}, nil
	}
	defer func() { loadLocalDataForCollector = oldLoadLocalData }()

	c := NewCollector()
	c.data = DevinData{Model: "swe-1-6", InputTokens: 29006, OutputTokens: 728}
	c.localSession = "old"

	session = "new"
	data := c.GetData().(DevinData)
	if data.Model != "swe-1-6" || data.InputTokens != 29006 || data.OutputTokens != 728 {
		t.Fatalf("GetData reset unrelated proxy state: model=%s input=%d output=%d; want swe-1-6 29006 728",
			data.Model, data.InputTokens, data.OutputTokens)
	}

	content := []byte{
		0x3A, 0x0C,
		0x10, 0xE4, 0x7C,
		0x18, 0x87, 0x01,
		0x4A, 0x07, 's', 'w', 'e', '-', '1', '-', '6',
	}
	stats := []byte{
		0xE2, 0x01, 0x2F,
		0x12, 0x15,
		0x22, 0x05, 0x15, 0x00, 0x90, 0x79, 0x46,
		0x2A, 0x0C, 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x12, 0x16,
		0x22, 0x05, 0x15, 0x00, 0x00, 0x07, 0x43,
		0x2A, 0x0D, 'o', 'u', 't', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
	}
	endMarker := []byte{0x78, 0x00}

	c.handleChatMessage(connectStream(content, stats, endMarker))
	data = c.GetData().(DevinData)
	if data.Model != "swe-1-6" || data.InputTokens != 16107 || data.OutputTokens != 135 {
		t.Fatalf("after new lower-token session: model=%s input=%d output=%d; want swe-1-6 16107 135",
			data.Model, data.InputTokens, data.OutputTokens)
	}
}

func TestCachedInputTokensCountTowardContext(t *testing.T) {
	contentMsg := []byte{
		0x3A, 0x0E,
		0x10, 0x15,
		0x18, 0x2F,
		0x4A, 0x09, 'k', 'i', 'm', 'i', '-', 'k', '2', '-', '6',
	}
	stats := []byte{
		0xE2, 0x01, 0x8C, 0x01,
		0x0A, 0x0B, 'T', 'o', 'k', 'e', 'n', ' ', 'U', 's', 'a', 'g', 'e',
		0x12, 0x23,
		0x2A, 0x0C, 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x22, 0x13,
		0x0A, 0x0C, 'I', 'n', 'p', 'u', 't', ' ', 't', 'o', 'k', 'e', 'n', 's',
		0x15, 0x00, 0x00, 0xA8, 0x41,
		0x12, 0x25,
		0x2A, 0x0D, 'o', 'u', 't', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x22, 0x14,
		0x0A, 0x0D, 'O', 'u', 't', 'p', 'u', 't', ' ', 't', 'o', 'k', 'e', 'n', 's',
		0x15, 0x00, 0x00, 0x3C, 0x42,
		0x12, 0x31,
		0x2A, 0x13, 'c', 'a', 'c', 'h', 'e', 'd', '_', 'i', 'n', 'p', 'u', 't', '_', 't', 'o', 'k', 'e', 'n', 's',
		0x22, 0x1A,
		0x0A, 0x13, 'C', 'a', 'c', 'h', 'e', 'd', ' ', 'i', 'n', 'p', 'u', 't', ' ', 't', 'o', 'k', 'e', 'n', 's',
		0x15, 0x00, 0x4C, 0x64, 0x46,
	}
	endMarker := []byte{0x78, 0x00}

	c := NewCollector()
	c.handleChatMessage(connectStream(contentMsg, stats, endMarker))
	data := c.GetData().(DevinData)
	if data.Model != "kimi-k2-6" || data.InputTokens != 14679 || data.OutputTokens != 47 {
		t.Fatalf("with cached input: model=%s input=%d output=%d; want kimi-k2-6 14679 47",
			data.Model, data.InputTokens, data.OutputTokens)
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
