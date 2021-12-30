package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
}

type docType = int

const (
	docTypeUnknown docType = iota
	docTypePhoto
	docTypeVideo
)

func NewApp(logger *zap.Logger) *App {
	return &App{
		logger: logger,
	}
}

func (a *App) Run() error {
	if err := a.run(context.Background(), a.logger); err != nil {
		return err
	}

	return nil
}

func (a *App) run(ctx context.Context, log *zap.Logger) error {
	dispatcher := tg.NewUpdateDispatcher()
	opts := telegram.Options{
		Logger:        log,
		UpdateHandler: dispatcher,
	}
	return telegram.BotFromEnvironment(ctx, opts, func(ctx context.Context, client *telegram.Client) error {
		api := tg.NewClient(client)
		u := uploader.NewUploader(api)
		sender := message.NewSender(api).WithUploader(u)

		h := &messageHandler{
			api:      api,
			uploader: u,
			sender:   sender,
		}

		dispatcher.OnNewMessage(h.onNewMessage)
		dispatcher.OnNewChannelMessage(h.onNewChannelMessage)

		return nil
	}, telegram.RunUntilCanceled)
}

type messageHandler struct {
	api      *tg.Client
	uploader *uploader.Uploader
	sender   *message.Sender
}

func (h *messageHandler) onNewMessage(ctx context.Context, entities tg.Entities, update *tg.UpdateNewMessage) error {
	return h.documentToMedia(ctx, entities, update)
}

func (h *messageHandler) onNewChannelMessage(ctx context.Context, entities tg.Entities, update *tg.UpdateNewChannelMessage) error {
	return h.documentToMedia(ctx, entities, update)
}

func (h *messageHandler) documentToMedia(ctx context.Context, entities tg.Entities, update message.AnswerableMessageUpdate) (err error) {
	m, ok := update.GetMessage().(*tg.Message)
	if !ok || m.Out {
		// ignore outgoing message
		return nil
	}

	target := h.sender.To(GetPeer(m.GetPeerID(), entities))
	if target == nil {
		return nil
	}

	defer func() {
		recover()

		_ = target.TypingAction().Cancel(ctx)

		if err != nil {
			_, _ = h.sender.Reply(entities, update).Text(ctx, fmt.Sprintf("failed with error: %+v", err))
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
	if dt == docTypeUnknown {
		return fmt.Errorf("unsupported content type '%s'", doc.GetMimeType())
	}

	down := downloader.NewDownloader()

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tmpDir)

	filePath := path.Join(tmpDir, getDocFileName(doc))
	defer os.Remove(filePath)

	_ = target.TypingAction().Typing(ctx)

	// Downloading gif to gifPath.
	loc := doc.AsInputDocumentFileLocation()
	if _, err := down.Download(h.api, loc).ToPath(ctx, filePath); err != nil {
		return err
	}

	f, err := h.uploader.FromPath(ctx, filePath)
	if err != nil {
		return fmt.Errorf("upload %q: %w", filePath, err)
	}

	var md message.MediaOption
	if dt == docTypePhoto {
		_ = target.TypingAction().UploadPhoto(ctx, 99)
		md = message.UploadedPhoto(f, styling.Plain(m.Message))
	} else {
		// нужно как-то добавить thumbnail
		_ = target.TypingAction().UploadVideo(ctx, 99)
		md = message.Video(f, styling.Plain(m.Message))
	}

	_, err = target.Media(ctx, md)
	if err != nil {
		return err
	}

	if _, err := target.Revoke().Messages(ctx, m.ID); err != nil {
		return err
	}

	return nil
}

func getDocumentType(doc *tg.Document) docType {
	switch doc.GetMimeType() {
	case "video/mp4", "video/quicktime":
		return docTypeVideo
	case "image/jpg", "image/jpeg", "image/png", "image/gif":
		return docTypePhoto
	}

	return docTypeUnknown
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
