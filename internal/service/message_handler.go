package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/far4599/telegram-lossless-media-bot/internal/model"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type MessageHandler interface {
	OnNewMessage(context.Context, tg.Entities, *tg.UpdateNewMessage) error
	OnNewChannelMessage(context.Context, tg.Entities, *tg.UpdateNewChannelMessage) error
}

type messageHandler struct {
	api    *tg.Client
	logger *zap.Logger
}

func NewMessageHandler(a *tg.Client, l *zap.Logger) *messageHandler {
	return &messageHandler{
		api:    a,
		logger: l,
	}
}

// OnNewMessage is a handler for dialogs
func (h *messageHandler) OnNewMessage(ctx context.Context, entities tg.Entities, update *tg.UpdateNewMessage) error {
	return h.documentToMedia(ctx, entities, update)
}

// OnNewChannelMessage is a handler for channels
func (h *messageHandler) OnNewChannelMessage(ctx context.Context, entities tg.Entities, update *tg.UpdateNewChannelMessage) error {
	return h.documentToMedia(ctx, entities, update)
}

// documentToMedia converts file from document message into photo/video message
func (h *messageHandler) documentToMedia(ctx context.Context, entities tg.Entities, update message.AnswerableMessageUpdate) (err error) {
	u := uploader.NewUploader(h.api)
	s := message.NewSender(h.api).WithUploader(u)

	m, ok := update.GetMessage().(*tg.Message)
	if !ok || m.Out {
		// ignore outgoing message
		return nil
	}

	target := s.To(GetPeer(m.GetPeerID(), entities))
	if target == nil {
		return nil
	}

	defer func() {
		recover()

		_ = target.TypingAction().Cancel(ctx)

		if err != nil {
			h.logger.Error("request failed", zap.Error(err))
			_, _ = s.Reply(entities, update).Text(ctx, fmt.Sprintf("failed with error: %+v", err))
		}
	}()

	media, ok := m.Media.(*tg.MessageMediaDocument)
	if !ok {
		// ignore not document
		return nil
	}

	d, ok := media.GetDocument()
	if !ok {
		return nil
	}

	doc, ok := d.(*tg.Document)
	if !ok {
		return nil
	}

	dt := getDocumentType(doc)
	if dt == model.DocTypeUnknown {
		return fmt.Errorf("unsupported content type '%s'", doc.GetMimeType())
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "failed to create tmp folder")
	}
	defer os.Remove(tmpDir)

	fileName := getDocFileName(doc)
	filePath := path.Join(tmpDir, fileName)
	if filePath == "" {
		return fmt.Errorf("file name is empty")
	}
	defer os.Remove(filePath)

	h.logger.Info("received document", zap.String("document_type", dt.String()), zap.String("file_name", fileName), zap.String("peer_id", m.GetPeerID().String()))

	_ = target.TypingAction().Typing(ctx)

	_, err = downloader.NewDownloader().Download(h.api, doc.AsInputDocumentFileLocation()).ToPath(ctx, filePath)
	if err != nil {
		return err
	}

	uploaderProgress := model.NewUploaderProgress()
	defer uploaderProgress.Close()

	go func() {
		for progress := range uploaderProgress.ProgressChan() {
			h.logger.Debug("upload progress changed", zap.Int32("progress", progress), zap.String("filePath", filePath))
			if dt == model.DocTypePhoto {
				_ = target.TypingAction().UploadPhoto(ctx, int(progress))
			} else {
				_ = target.TypingAction().UploadVideo(ctx, int(progress))
			}
		}
	}()

	f, err := u.WithProgress(uploaderProgress).FromPath(ctx, filePath)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to upload %s", fileName))
	}

	var md message.MediaOption
	if dt == model.DocTypePhoto {
		md = message.UploadedPhoto(f, styling.Plain(m.Message))
	} else {
		// нужно как-то добавить thumbnail
		md = message.Video(f, styling.Plain(m.Message))
	}

	_, err = target.Media(ctx, md)
	if err != nil {
		return err
	}

	_, err = target.Revoke().Messages(ctx, m.ID)
	if err != nil {
		return err
	}

	return nil
}

func getDocumentType(doc *tg.Document) model.DocType {
	switch doc.GetMimeType() {
	case "video/mp4", "video/quicktime":
		return model.DocTypeVideo
	case "image/jpg", "image/jpeg", "image/png", "image/gif":
		return model.DocTypePhoto
	}

	return model.DocTypeUnknown
}

func getDocFileName(doc *tg.Document) string {
	for _, attr := range doc.GetAttributes() {
		filename, ok := attr.(*tg.DocumentAttributeFilename)
		if ok {
			return filename.GetFileName()
		}
	}
	return ""
}

func GetPeer(pc tg.PeerClass, entities tg.Entities) tg.InputPeerClass {
	switch pc.(type) {
	case *tg.PeerUser:
		u := pc.(*tg.PeerUser)
		return entities.Users[u.GetUserID()].AsInputPeer()
	case *tg.PeerChat:
		c := pc.(*tg.PeerChat)
		return entities.Chats[c.GetChatID()].AsInputPeer()
	case *tg.PeerChannel:
		c := pc.(*tg.PeerChannel)
		return entities.Channels[c.GetChannelID()].AsInputPeer()
	}

	return nil
}
