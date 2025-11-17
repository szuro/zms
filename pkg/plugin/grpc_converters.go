package plugin

import (
	"encoding/json"

	"zms.szuro.net/pkg/proto"
	"zms.szuro.net/pkg/zbx"
)

// protoHistoryToZbx converts proto.History to zbx.History.
func protoHistoryToZbx(protoHistory []*proto.History) []zbx.History {
	result := make([]zbx.History, 0, len(protoHistory))

	for _, ph := range protoHistory {
		h := zbx.History{
			ItemID: ph.Itemid,
			Name:   ph.Name,
			Clock:  ph.Clock,
			Groups: ph.Groups,
			Ns:     ph.Ns,
			Tags:   protoTagsToZbx(ph.Tags),
			Type:   int32(ph.ValueType),
		}

		// Convert host
		if ph.Host != nil {
			h.Host = &zbx.Host{
				Host: ph.Host.Host,
				Name: ph.Host.Name,
			}
		}

		// Convert value based on type
		switch ph.Value.(type) {
		case *proto.History_NumericValue:
			h.Value = ph.GetNumericValue()
		case *proto.History_StringValue:
			h.Value = ph.GetStringValue()
		}

		// Log-specific fields
		if ph.ValueType == proto.ValueType_LOG {
			h.Timestamp = ph.Timestamp
			h.Source = ph.Source
			h.Severity = int32(ph.Severity)
			h.EventID = ph.Eventid
		}

		result = append(result, h)
	}

	return result
}

// protoTrendsToZbx converts proto.Trend to zbx.Trend.
func protoTrendsToZbx(protoTrends []*proto.Trend) []zbx.Trend {
	result := make([]zbx.Trend, 0, len(protoTrends))

	for _, pt := range protoTrends {
		t := zbx.Trend{
			ItemID: pt.Itemid,
			Name:   pt.Name,
			Clock:  pt.Clock,
			Count:  pt.Count,
			Groups: pt.Groups,
			Min:    pt.Min,
			Max:    pt.Max,
			Avg:    pt.Avg,
			Tags:   protoTagsToZbx(pt.Tags),
			Type:   int32(pt.ValueType),
		}

		// Convert host
		if pt.Host != nil {
			t.Host = &zbx.Host{
				Host: pt.Host.Host,
				Name: pt.Host.Name,
			}
		}

		result = append(result, t)
	}

	return result
}

// protoEventsToZbx converts proto.Event to zbx.Event.
func protoEventsToZbx(protoEvents []*proto.Event) []zbx.Event {
	result := make([]zbx.Event, 0, len(protoEvents))

	for _, pe := range protoEvents {
		e := zbx.Event{
			Clock:    pe.Clock,
			NS:       pe.Ns,
			Value:    int32(pe.Value),
			EventID:  pe.Eventid,
			PEventID: pe.PEventid,
			Name:     pe.Name,
			Severity: int32(pe.Severity),
			Groups:   pe.Groups,
			Tags:     protoTagsToZbx(pe.Tags),
		}

		// Convert hosts
		if len(pe.Hosts) > 0 {
			e.Hosts = make([]zbx.Host, 0, len(pe.Hosts))
			for _, ph := range pe.Hosts {
				e.Hosts = append(e.Hosts, zbx.Host{
					Host: ph.Host,
					Name: ph.Name,
				})
			}
		}

		result = append(result, e)
	}

	return result
}

// protoTagsToZbx converts proto.Tag to zbx.Tag.
func protoTagsToZbx(protoTags []*proto.Tag) []zbx.Tag {
	if len(protoTags) == 0 {
		return nil
	}

	result := make([]zbx.Tag, 0, len(protoTags))
	for _, pt := range protoTags {
		result = append(result, zbx.Tag{
			Tag:   pt.Tag,
			Value: pt.Value,
		})
	}

	return result
}

// ZbxHistoryToProto converts a single zbx.History to proto.History.
func ZbxHistoryToProto(h *zbx.History) *proto.History {
	ph := &proto.History{
		Itemid:    h.ItemID,
		Name:      h.Name,
		Clock:     h.Clock,
		Groups:    h.Groups,
		Ns:        h.Ns,
		Tags:      zbxTagsToProto(h.Tags),
		ValueType: proto.ValueType(h.Type),
	}

	// Convert host
	if h.Host != nil {
		ph.Host = &proto.Host{
			Host: h.Host.Host,
			Name: h.Host.Name,
		}
	}

	// Convert value based on type
	switch v := h.Value.(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			ph.Value = &proto.History_NumericValue{NumericValue: f}
		} else {
			ph.Value = &proto.History_StringValue{StringValue: v.String()}
		}
	case string:
		ph.Value = &proto.History_StringValue{StringValue: v}
	case float64:
		ph.Value = &proto.History_NumericValue{NumericValue: v}
	case int, int32, int64:
		ph.Value = &proto.History_NumericValue{NumericValue: float64(v.(int64))}
	}

	// Log-specific fields
	if h.Type == zbx.LOG {
		ph.Timestamp = h.Timestamp
		ph.Source = h.Source
		ph.Severity = proto.Severity(h.Severity)
		ph.Eventid = h.EventID
	}

	return ph
}

// ZbxHistorySliceToProto converts zbx.History slice to proto.History slice.
func ZbxHistorySliceToProto(zbxHistory []zbx.History) []*proto.History {
	result := make([]*proto.History, 0, len(zbxHistory))
	for _, h := range zbxHistory {
		result = append(result, ZbxHistoryToProto(&h))
	}
	return result
}

// ZbxTrendToProto converts a single zbx.Trend to proto.Trend.
func ZbxTrendToProto(t *zbx.Trend) *proto.Trend {
	pt := &proto.Trend{
		Itemid:    t.ItemID,
		Name:      t.Name,
		Clock:     t.Clock,
		Count:     t.Count,
		Groups:    t.Groups,
		Min:       t.Min,
		Max:       t.Max,
		Avg:       t.Avg,
		Tags:      zbxTagsToProto(t.Tags),
		ValueType: proto.ValueType(t.Type),
	}

	// Convert host
	if t.Host != nil {
		pt.Host = &proto.Host{
			Host: t.Host.Host,
			Name: t.Host.Name,
		}
	}

	return pt
}

// ZbxTrendsToProto converts zbx.Trend slice to proto.Trend slice.
func ZbxTrendsToProto(zbxTrends []zbx.Trend) []*proto.Trend {
	result := make([]*proto.Trend, 0, len(zbxTrends))
	for _, t := range zbxTrends {
		result = append(result, ZbxTrendToProto(&t))
	}
	return result
}

// ZbxEventToProto converts a single zbx.Event to proto.Event.
func ZbxEventToProto(e *zbx.Event) *proto.Event {
	pe := &proto.Event{
		Clock:    e.Clock,
		Ns:       e.NS,
		Value:    proto.EventValue(e.Value),
		Eventid:  e.EventID,
		PEventid: e.PEventID,
		Name:     e.Name,
		Severity: proto.Severity(e.Severity),
		Groups:   e.Groups,
		Tags:     zbxTagsToProto(e.Tags),
	}

	// Convert hosts
	if len(e.Hosts) > 0 {
		pe.Hosts = make([]*proto.Host, 0, len(e.Hosts))
		for _, h := range e.Hosts {
			pe.Hosts = append(pe.Hosts, &proto.Host{
				Host: h.Host,
				Name: h.Name,
			})
		}
	}

	return pe
}

// ZbxEventsToProto converts zbx.Event slice to proto.Event slice.
func ZbxEventsToProto(zbxEvents []zbx.Event) []*proto.Event {
	result := make([]*proto.Event, 0, len(zbxEvents))
	for _, e := range zbxEvents {
		result = append(result, ZbxEventToProto(&e))
	}
	return result
}

// zbxTagsToProto converts zbx.Tag to proto.Tag.
func zbxTagsToProto(zbxTags []zbx.Tag) []*proto.Tag {
	if len(zbxTags) == 0 {
		return nil
	}

	result := make([]*proto.Tag, 0, len(zbxTags))
	for _, t := range zbxTags {
		result = append(result, &proto.Tag{
			Tag:   t.Tag,
			Value: t.Value,
		})
	}

	return result
}

// StringToExportType converts export type string to proto.ExportType.
func StringToExportType(s string) proto.ExportType {
	switch s {
	case zbx.HISTORY:
		return proto.ExportType_HISTORY
	case zbx.TREND:
		return proto.ExportType_TRENDS
	case zbx.EVENT:
		return proto.ExportType_EVENTS
	default:
		return proto.ExportType_HISTORY
	}
}
