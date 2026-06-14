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

func TestHandleChatMessageIgnoresSubagents(t *testing.T) {
	stats, err := hex.DecodeString("0a28626f742d35356662346638312d326461652d343833312d623861362d646336323233623333616531120c08f78bbcd10610a1d1a8de01e2016e0a13526573706f6e73652053746174697374696373123c2a0e6167656e745f6d65737361676573222a0a0e4167656e74206d65737361676573150000803f1a08206d6573736167652209206d6573736167657312192a056d6f64656c1a100a054d6f64656c12075357452d312e36e201ba010a0b546f6b656e20557361676512342a0c696e7075745f746f6b656e7322240a0c496e70757420746f6b656e731500587a461a0620746f6b656e220720746f6b656e7312362a0d6f75747075745f746f6b656e7322250a0d4f757470757420746f6b656e73150000ef431a0620746f6b656e220720746f6b656e73123d2a136361636865645f696e7075745f746f6b656e7322260a1343616368656420696e70757420746f6b656e731a0620746f6b656e220720746f6b656e73")
	if err != nil {
		t.Fatal(err)
	}

	c := NewCollector()

	// Main session response with stats — should be captured.
	c.handleChatMessage(connectStream([]byte{}, stats))
	data := c.GetData().(DevinData)
	if data.InputTokens != 16022 || data.OutputTokens != 478 {
		t.Fatalf("main session: collector data = %d, %d; want 16022, 478", data.InputTokens, data.OutputTokens)
	}

	// Subagent response without stats — should be ignored, data unchanged.
	c.handleChatMessage(connectStream([]byte{}, []byte{0x3a, 0x04, 0x10, 0x31, 0x18, 0x3e}))
	data = c.GetData().(DevinData)
	if data.InputTokens != 16022 || data.OutputTokens != 478 {
		t.Fatalf("after subagent: collector data = %d, %d; want 16022, 478", data.InputTokens, data.OutputTokens)
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
