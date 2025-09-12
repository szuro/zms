package input

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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	zbxpkg "szuro.net/zms/pkg/zbx"
	"szuro.net/zms/internal/config"
	"szuro.net/zms/internal/logger"
)

var (
	ndjsonLinesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zms_http_ndjson_lines_total",
			Help: "Total number of NDJSON lines received per endpoint",
		},
		[]string{"endpoint"},
	)

	ndjsonParseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zms_http_ndjson_parse_errors_total",
			Help: "Total number of NDJSON parse errors per endpoint",
		},
		[]string{"endpoint"},
	)
)

type HTTPInput struct {
	baseInput
}

func NewHTTPInput(zmsConf config.ZMSConf) (*HTTPInput, error) {
	hi := &HTTPInput{
		baseInput{
			config:   zmsConf,
			subjects: make(map[string]Subjecter),
		},
	}
	historySubject := NewSubject[zbxpkg.History]()
	historySubject.Funnel = make(chan any, zmsConf.BufferSize*2)
	historySubject.SetBuffer(zmsConf.BufferSize)
	hi.subjects[zbxpkg.HISTORY] = &historySubject

	eventSubject := NewSubject[zbxpkg.Event]()
	eventSubject.Funnel = make(chan any, zmsConf.BufferSize*2)
	eventSubject.SetBuffer(zmsConf.BufferSize)
	hi.subjects[zbxpkg.EVENT] = &eventSubject

	return hi, nil
}

func (hi *HTTPInput) Prepare() {
	hi.baseInput.Prepare()
}

func (hi *HTTPInput) Start() {
	http.HandleFunc("/history", hi.handleHistory)
	http.HandleFunc("/events", hi.handleEvents)
	ndjsonLinesReceived.WithLabelValues(zbxpkg.HISTORY).Add(0)
	ndjsonLinesReceived.WithLabelValues(zbxpkg.EVENT).Add(0)
	ndjsonParseErrors.WithLabelValues(zbxpkg.HISTORY).Add(0)
	ndjsonParseErrors.WithLabelValues(zbxpkg.EVENT).Add(0)
	hi.baseInput.Start()
}

func (hi *HTTPInput) Stop() error {
	return hi.baseInput.Stop()
}

func (hi *HTTPInput) IsReady() bool {
	return true // HTTP server is always ready after Start
}

func (hi *HTTPInput) handleHistory(w http.ResponseWriter, r *http.Request) {
	hi.handleNDJSON(w, r, func(line string) {
		var hExport zbxpkg.History
		if err := json.Unmarshal([]byte(line), &hExport); err != nil {
			logger.Error("Failed to parse history line", slog.Any("error", err))
			ndjsonParseErrors.WithLabelValues(zbxpkg.HISTORY).Inc()
			return
		}
		ndjsonLinesReceived.WithLabelValues(zbxpkg.HISTORY).Inc()
		subject, ok := hi.subjects[zbxpkg.HISTORY]
		if !ok {
			logger.Error("No subject!")
			return
		}
		funnel := subject.GetFunnel()
		if funnel == nil {
			logger.Error("No funnel for HISTORY export", slog.String("subject", zbxpkg.HISTORY))
			return
		}
		funnel <- hExport
	})
}

func (hi *HTTPInput) handleEvents(w http.ResponseWriter, r *http.Request) {
	hi.handleNDJSON(w, r, func(line string) {
		var eExport zbxpkg.Event
		if err := json.Unmarshal([]byte(line), &eExport); err != nil {
			logger.Error("Failed to parse event line", slog.Any("error", err))
			ndjsonParseErrors.WithLabelValues(zbxpkg.EVENT).Inc()
			return
		}
		ndjsonLinesReceived.WithLabelValues(zbxpkg.EVENT).Inc()
		subject, ok := hi.subjects[zbxpkg.EVENT]
		if !ok {
			logger.Error("No subject!")
			return
		}
		funnel := subject.GetFunnel()
		if funnel == nil {
			logger.Error("No funnel for EVENT export", slog.String("subject", zbxpkg.EVENT))
			return
		}
		funnel <- eExport
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
				logger.Error("Failed to create gzip reader", slog.Any("error", err))
				return
			}
			defer gz.Close()
			bodyReader = gz
		case "deflate":
			zr, err := zlib.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				logger.Error("Failed to create zlib/deflate reader", slog.Any("error", err))
				return
			}
			defer zr.Close()
			bodyReader = zr
		case "zstd", "ztsd":
			zr, err := zstd.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				logger.Error("Failed to create zstd reader", slog.Any("error", err))
				return
			}
			defer zr.Close()
			bodyReader = zr
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			logger.Error("Unsupported Content-Encoding", slog.String("encoding", ce))
			return
		}
	}

	reader := bufio.NewReader(bodyReader)
	fin := false
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			fin = true
		}
		if err != nil && err != io.EOF {
			logger.Error("Error reading request body", slog.Any("error", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		handleLine(line)
		if fin {
			break
		}
	}
	w.WriteHeader(http.StatusOK)
}
