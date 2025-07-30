package claude

// mockStreamer is a test double for the Streamer interface
type mockStreamer struct {
	startCalls    []streamCall
	contentCalls  []streamCall
	endCalls      []streamCall
	streamErr     error // Can be set to simulate streaming failures
}

type streamCall struct {
	runID     string
	messageID string
	content   string
}

func (m *mockStreamer) StreamStart(runID, messageID string) error {
	m.startCalls = append(m.startCalls, streamCall{runID: runID, messageID: messageID})
	return m.streamErr
}

func (m *mockStreamer) StreamContent(runID, messageID, content string) error {
	m.contentCalls = append(m.contentCalls, streamCall{runID: runID, messageID: messageID, content: content})
	return m.streamErr
}

func (m *mockStreamer) StreamEnd(runID, messageID string) error {
	m.endCalls = append(m.endCalls, streamCall{runID: runID, messageID: messageID})
	return m.streamErr
}