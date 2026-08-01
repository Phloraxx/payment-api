package reconciliation

import "testing"

func FuzzDelimitedStatementParserNeverPanics(f *testing.F) {
	f.Add([]byte("Date,Credit,Narration,RRN\n01/08/2026,100.01,UPI credit,123456789012\n"))
	f.Add([]byte("\xff\x00\x01"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256<<10 {
			t.Skip()
		}
		_, _ = parseDelimited(data)
	})
}
