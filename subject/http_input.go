package subject

import (
	"bufio"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms"
)

type HTTPInput struct {
	baseInput
}

func NewHTTPInput(zmsConf zms.ZMSConf) (*HTTPInput, error) {
	hi := &HTTPInput{
		baseInput{
			config:   zmsConf,
			subjects: make(map[string]Subjecter),
		},
	}
	historySubject := NewSubject[zbx.History]()
	historySubject.Funnel = make(chan any, zmsConf.BufferSize)
	historySubject.SetBuffer(zmsConf.BufferSize)
	hi.subjects[zbx.HISTORY] = &historySubject

	eventSubject := NewSubject[zbx.Event]()
	eventSubject.Funnel = make(chan any, zmsConf.BufferSize)
	eventSubject.SetBuffer(zmsConf.BufferSize)
	hi.subjects[zbx.EVENT] = &eventSubject

	return hi, nil
}

func (hi *HTTPInput) Prepare() {
	hi.baseInput.Prepare()
}

func (hi *HTTPInput) Start() {
	http.HandleFunc("/history", hi.handleHistory)
	http.HandleFunc("/events", hi.handleEvents)
	hi.baseInput.Start()
}

func (hi *HTTPInput) Stop() {
	hi.baseInput.Stop()
}

func (hi *HTTPInput) IsReady() bool {
	return true // HTTP server is always ready after Start
}

func (hi *HTTPInput) handleHistory(w http.ResponseWriter, r *http.Request) {
	hi.handleNDJSON(w, r, func(line string) {
		var hExport zbx.History
		if err := json.Unmarshal([]byte(line), &hExport); err != nil {
			slog.Error("Failed to parse history line", slog.Any("error", err))
			return
		}
		subject, ok := hi.subjects[zbx.HISTORY]
		if !ok {
			slog.Error("No subject!")
		}
		funnel := subject.GetFunnel()
		if funnel == nil {
			slog.Error("No funnel for HISTORY export", slog.Any("subject", zbx.HISTORY))
			return
		}
		select {
		case funnel <- hExport:
		default:
			slog.Warn("Funnel for HISTORY is full, dropping data")
		}
	})
}

func (hi *HTTPInput) handleEvents(w http.ResponseWriter, r *http.Request) {
	hi.handleNDJSON(w, r, func(line string) {
		var eExport zbx.Event
		if err := json.Unmarshal([]byte(line), &eExport); err != nil {
			slog.Error("Failed to parse event line", slog.Any("error", err))
			return
		}
		subject, ok := hi.subjects[zbx.EVENT]
		if !ok {
			slog.Error("No subject!")
		}
		funnel := subject.GetFunnel()
		if funnel == nil {
			slog.Error("No funnel for EVENT export", slog.Any("subject", zbx.EVENT))
			return
		}
		select {
		case funnel <- eExport:
		default:
			slog.Warn("Funnel for EVENT is full, dropping data")
		}
	})
}

// handleNDJSON handles decompression, NDJSON reading, and error responses for HTTPInput
func (hi *HTTPInput) handleNDJSON(w http.ResponseWriter, r *http.Request, handleLine func(string)) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var bodyReader io.Reader = r.Body
	if ce := r.Header.Get("Content-Encoding"); ce != "" {
		switch strings.ToLower(ce) {
		case "gzip":
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				slog.Error("Failed to create gzip reader", slog.Any("error", err))
				return
			}
			defer gz.Close()
			bodyReader = gz
		case "deflate":
			zr, err := zlib.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				slog.Error("Failed to create zlib/deflate reader", slog.Any("error", err))
				return
			}
			defer zr.Close()
			bodyReader = zr
		case "zstd", "ztsd":
			zr, err := zstd.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				slog.Error("Failed to create zstd reader", slog.Any("error", err))
				return
			}
			defer zr.Close()
			bodyReader = zr
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			slog.Error("Unsupported Content-Encoding", slog.Any("encoding", ce))
			return
		}
	}

	reader := bufio.NewReader(bodyReader)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		handleLine(line)
	}
	w.WriteHeader(http.StatusOK)
}
