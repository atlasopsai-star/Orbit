package internal

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/metadata"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/notify"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/processbar"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/spferror"
)

type ModelUpdateMessage interface {
	ApplyToModel(m *model) tea.Cmd
	GetReqID() int
}

type BaseMessage struct {
	reqID         int
	affectedPaths []string
}

func copyAffectedPaths(paths []string) []string {
	return append([]string(nil), paths...)
}

func (msg BaseMessage) GetReqID() int {
	return msg.reqID
}

type PasteOperationMsg struct {
	BaseMessage

	state processbar.ProcessState
}

func NewPasteOperationMsg(state processbar.ProcessState, reqID int, affectedPaths ...string) PasteOperationMsg {
	return PasteOperationMsg{
		state: state,
		BaseMessage: BaseMessage{
			reqID:         reqID,
			affectedPaths: copyAffectedPaths(affectedPaths),
		},
	}
}

func (msg PasteOperationMsg) ApplyToModel(m *model) tea.Cmd {
	if (msg.state == processbar.Failed || msg.state == processbar.Successful) && m.clipboard.IsCut() {
		m.clipboard.Reset(false)
	}
	return m.invalidateFileOperation(msg.affectedPaths)
}

type CreateOperationMsg struct {
	BaseMessage

	state processbar.ProcessState
}

func NewCreateOperationMsg(state processbar.ProcessState, reqID int, affectedPaths ...string) CreateOperationMsg {
	return CreateOperationMsg{
		state: state,
		BaseMessage: BaseMessage{
			reqID:         reqID,
			affectedPaths: copyAffectedPaths(affectedPaths),
		},
	}
}

func (msg CreateOperationMsg) ApplyToModel(m *model) tea.Cmd {
	return m.invalidateFileOperation(msg.affectedPaths)
}

type DeleteOperationMsg struct {
	BaseMessage

	state processbar.ProcessState
}

func NewDeleteOperationMsg(state processbar.ProcessState, reqID int, affectedPaths ...string) DeleteOperationMsg {
	return DeleteOperationMsg{
		state: state,
		BaseMessage: BaseMessage{
			reqID:         reqID,
			affectedPaths: copyAffectedPaths(affectedPaths),
		},
	}
}

func (msg DeleteOperationMsg) ApplyToModel(m *model) tea.Cmd {
	// Remove selection
	m.getFocusedFilePanel().ResetSelected()
	return m.invalidateFileOperation(msg.affectedPaths)
}

type ProcessBarUpdateMsg struct {
	BaseMessage

	pMsg processbar.UpdateMsg
}

func (msg ProcessBarUpdateMsg) ApplyToModel(m *model) tea.Cmd {
	cmd, err := msg.pMsg.Apply(&m.processBarModel)
	if err != nil {
		slog.Error("Error applying processbar update", "error", err)
	}
	return processCmdToTeaCmd(cmd)
}

type CompressOperationMsg struct {
	BaseMessage

	state processbar.ProcessState
}

func NewCompressOperationMsg(state processbar.ProcessState, reqID int, affectedPaths ...string) CompressOperationMsg {
	return CompressOperationMsg{
		state: state,
		BaseMessage: BaseMessage{
			reqID:         reqID,
			affectedPaths: copyAffectedPaths(affectedPaths),
		},
	}
}

func (msg CompressOperationMsg) ApplyToModel(m *model) tea.Cmd {
	return m.invalidateFileOperation(msg.affectedPaths)
}

type ExtractOperationMsg struct {
	BaseMessage

	state processbar.ProcessState
}

func NewExtractOperationMsg(state processbar.ProcessState, reqID int, affectedPaths ...string) ExtractOperationMsg {
	return ExtractOperationMsg{
		state: state,
		BaseMessage: BaseMessage{
			reqID:         reqID,
			affectedPaths: copyAffectedPaths(affectedPaths),
		},
	}
}

func (msg ExtractOperationMsg) ApplyToModel(m *model) tea.Cmd {
	return m.invalidateFileOperation(msg.affectedPaths)
}

type MetadataMsg struct {
	BaseMessage

	meta            metadata.Metadata
	metadataFocused bool
}

func NewMetadataMsg(meta metadata.Metadata, metadataFocused bool, reqID int) MetadataMsg {
	return MetadataMsg{
		meta:            meta,
		metadataFocused: metadataFocused,
		BaseMessage: BaseMessage{
			reqID: reqID,
		},
	}
}

func (msg MetadataMsg) ApplyToModel(m *model) tea.Cmd {
	m.fileMetaData.SetMetadataCache(msg.meta, msg.metadataFocused)
	selectedItem := m.getFocusedFilePanel().GetFocusedItemPtr()
	if selectedItem == nil {
		slog.Debug("Panel empty or cursor invalid. Ignoring MetadataMsg")
		return nil
	}
	if selectedItem.Location != msg.meta.GetPath() {
		slog.Debug("MetadataMsg for older files. Ignoring",
			"currentItem", selectedItem.Location, "msgItem", msg.meta.GetPath())
		return nil
	}
	if (m.focusPanel == metadataFocus) != msg.metadataFocused {
		slog.Debug("MetadataMsg for older state. Ignoring",
			"actualFocus", m.focusPanel, "msgFocus", msg.metadataFocused)
		return nil
	}
	m.fileMetaData.SetMetadata(msg.meta, msg.metadataFocused)
	return nil
}

type NotifyModalUpdateMsg struct {
	BaseMessage

	m notify.Model
}

func NewNotifyModalMsg(m notify.Model, reqID int) NotifyModalUpdateMsg {
	return NotifyModalUpdateMsg{
		m: m,
		BaseMessage: BaseMessage{
			reqID: reqID,
		},
	}
}

func (msg NotifyModalUpdateMsg) ApplyToModel(m *model) tea.Cmd {
	m.notifyModel = msg.m
	return nil
}

type SpfErrorModalUpdateMsg struct {
	BaseMessage

	m spferror.Model
}

func NewSpfErrorModalMsg(m spferror.Model, reqID int) SpfErrorModalUpdateMsg {
	return SpfErrorModalUpdateMsg{
		m: m,
		BaseMessage: BaseMessage{
			reqID: reqID,
		},
	}
}

func (msg SpfErrorModalUpdateMsg) ApplyToModel(m *model) tea.Cmd {
	m.spfError = msg.m
	return nil
}
