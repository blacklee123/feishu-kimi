package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type CardKind string
type CardChatType string

var (
	ClearCardKind        = CardKind("clear")            // 清空上下文
	PicModeChangeKind    = CardKind("pic_mode_change")  // 切换图片创作模式
	VisionModeChangeKind = CardKind("vision_mode")      // 切换图片解析模式
	PicResolutionKind    = CardKind("pic_resolution")   // 图片分辨率调整
	PicStyleKind         = CardKind("pic_style")        // 图片风格调整
	VisionStyleKind      = CardKind("vision_style")     // 图片推理级别调整
	PicTextMoreKind      = CardKind("pic_text_more")    // 重新根据文本生成图片
	PicVarMoreKind       = CardKind("pic_var_more")     // 变量图片
	RoleTagsChooseKind   = CardKind("role_tags_choose") // 内置角色所属标签选择
	RoleChooseKind       = CardKind("role_choose")      // 内置角色选择
	AIModeChooseKind     = CardKind("ai_mode_choose")   // AI模式选择
)

var (
	GroupChatType = CardChatType("group")
	UserChatType  = CardChatType("personal")
)

type CardMsg struct {
	Kind      CardKind
	ChatType  CardChatType
	Value     interface{}
	SessionId string
	MsgId     string
}

type MenuOption struct {
	value string
	label string
}

func (a *ActionInfo) retrieveUserInfo(ctx context.Context, userId string) (*larkcontact.User, error) {
	client := a.larkClient
	req := larkcontact.NewGetUserReqBuilder().
		UserIdType(`open_id`).
		UserId(userId).
		DepartmentIdType(`open_department_id`).
		Build()

	// 发起请求
	resp, err := client.Contact.User.Get(ctx, req)

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(resp.Msg)
	}
	return resp.Data.User, nil
}

func (a *ActionInfo) replyCard(ctx context.Context, msgId *string, cardContent string) error {
	client := a.larkClient
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Uuid(uuid.New().String()).
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		log.Printf("服务端错误 resp code[%v], msg [%v] requestId [%v] ", resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func newSendCard(
	header *larkcard.MessageCardHeader,
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Header(header).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

func newSimpleSendCard(
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

// withSplitLine 用于生成分割线
func withSplitLine() larkcard.MessageCardElement {
	splitLine := larkcard.NewMessageCardHr().
		Build()
	return splitLine
}

// withHeader 用于生成消息头
func withHeader(title string, color string) *larkcard.
	MessageCardHeader {
	if title == "" {
		title = "🤖️机器人提醒"
	}
	header := larkcard.NewMessageCardHeader().
		Template(color).
		Title(larkcard.NewMessageCardPlainText().
			Content(title).
			Build()).
		Build()
	return header
}

// withNote 用于生成纯文本脚注
func withNote(note string) larkcard.MessageCardElement {
	noteElement := larkcard.NewMessageCardNote().
		Elements([]larkcard.MessageCardNoteElement{larkcard.NewMessageCardPlainText().
			Content(note).
			Build()}).
		Build()
	return noteElement
}

func withImg(fileKey string, alt string) larkcard.MessageCardElement {
	mainElement := larkcard.NewMessageCardEmbedImage().
		ImgKey(fileKey).
		Alt(larkcard.NewMessageCardPlainText().Content(alt).Build()).
		Build()
	return mainElement
}

// withMainMd 用于生成markdown消息体
func withMainMd(msg string) larkcard.MessageCardElement {
	// fmt.Println("beforprocessMessage", msg)
	// msg, i := processMessage(msg)
	// fmt.Println("afterprocessMessage", msg)
	// msg = cleanTextBlock(msg)
	// fmt.Println("aftercleanTextBlock", msg)
	// if i != nil {
	// 	return nil
	// }
	mainElement := larkcard.NewMessageCardMarkdown().
		Content(msg).Build()
	return mainElement
}

// withMainText 用于生成纯文本消息体
func withMainText(msg string) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = cleanTextBlock(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardPlainText().
				Content(msg).
				Build()).
			IsShort(false).
			Build()}).
		Build()
	return mainElement
}

func withImageDiv(imageKey string) larkcard.MessageCardElement {
	imageElement := larkcard.NewMessageCardImage().
		ImgKey(imageKey).
		Alt(larkcard.NewMessageCardPlainText().Content("").
			Build()).
		Preview(true).
		Mode(larkcard.MessageCardImageModelCropCenter).
		CompactWidth(true).
		Build()
	return imageElement
}

// withMdAndExtraBtn 用于生成带有额外按钮的消息体
func withMdAndExtraBtn(msg string, btn *larkcard.
	MessageCardEmbedButton) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = processNewLine(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields(
			[]*larkcard.MessageCardField{
				larkcard.NewMessageCardField().
					Text(larkcard.NewMessageCardLarkMd().
						Content(msg).
						Build()).
					IsShort(true).
					Build()}).
		Extra(btn).
		Build()
	return mainElement
}

func newBtn(content string, value map[string]interface{},
	typename larkcard.MessageCardButtonType) *larkcard.
	MessageCardEmbedButton {
	btn := larkcard.NewMessageCardEmbedButton().
		Type(typename).
		Value(value).
		Text(larkcard.NewMessageCardPlainText().
			Content(content).
			Build())
	return btn
}

func newMenu(
	placeHolder string,
	value map[string]interface{},
	options ...MenuOption,
) *larkcard.
	MessageCardEmbedSelectMenuStatic {
	var aOptionPool []*larkcard.MessageCardEmbedSelectOption
	for _, option := range options {
		aOption := larkcard.NewMessageCardEmbedSelectOption().
			Value(option.value).
			Text(larkcard.NewMessageCardPlainText().
				Content(option.label).
				Build())
		aOptionPool = append(aOptionPool, aOption)

	}
	btn := larkcard.NewMessageCardEmbedSelectMenuStatic().
		MessageCardEmbedSelectMenuStatic(larkcard.NewMessageCardEmbedSelectMenuBase().
			Options(aOptionPool).
			Placeholder(larkcard.NewMessageCardPlainText().
				Content(placeHolder).
				Build()).
			Value(value).
			Build()).
		Build()
	return btn
}

// 清除卡片按钮
func withClearDoubleCheckBtn(sessionID *string) larkcard.MessageCardElement {
	confirmBtn := newBtn("确认清除", map[string]interface{}{
		"value":     "1",
		"kind":      ClearCardKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"value":     "0",
		"kind":      ClearCardKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withPicModeDoubleCheckBtn(sessionID *string) larkcard.
	MessageCardElement {
	confirmBtn := newBtn("切换模式", map[string]interface{}{
		"value":     "1",
		"kind":      PicModeChangeKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"value":     "0",
		"kind":      PicModeChangeKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}
func withVisionModeDoubleCheckBtn(sessionID *string) larkcard.
	MessageCardElement {
	confirmBtn := newBtn("切换模式", map[string]interface{}{
		"value":     "1",
		"kind":      VisionModeChangeKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"value":     "0",
		"kind":      VisionModeChangeKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withOneBtn(btn *larkcard.MessageCardEmbedButton) larkcard.
	MessageCardElement {
	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{btn}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withRoleTagsBtn(sessionID *string, tags ...string) larkcard.
	MessageCardElement {
	var menuOptions []MenuOption

	for _, tag := range tags {
		menuOptions = append(menuOptions, MenuOption{
			label: tag,
			value: tag,
		})
	}
	cancelMenu := newMenu("选择角色分类",
		map[string]interface{}{
			"value":     "0",
			"kind":      RoleTagsChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withRoleBtn(sessionID *string, titles ...string) larkcard.
	MessageCardElement {
	var menuOptions []MenuOption

	for _, tag := range titles {
		menuOptions = append(menuOptions, MenuOption{
			label: tag,
			value: tag,
		})
	}
	cancelMenu := newMenu("查看内置角色",
		map[string]interface{}{
			"value":     "0",
			"kind":      RoleChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withAIModeBtn(sessionID *string, aiModeStrs []string) larkcard.MessageCardElement {
	var menuOptions []MenuOption
	for _, label := range aiModeStrs {
		menuOptions = append(menuOptions, MenuOption{
			label: label,
			value: label,
		})
	}

	cancelMenu := newMenu("选择模式",
		map[string]interface{}{
			"value":     "0",
			"kind":      AIModeChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func (a *ActionInfo) replyMsg(ctx context.Context, msg string, msgId *string) error {
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := a.larkClient
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func (a *ActionInfo) uploadImage(base64Str string) (*string, error) {
	imageBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	client := a.larkClient
	resp, err := client.Im.Image.Create(context.Background(),
		larkim.NewCreateImageReqBuilder().
			Body(larkim.NewCreateImageReqBodyBuilder().
				ImageType(larkim.ImageTypeMessage).
				Image(bytes.NewReader(imageBytes)).
				Build()).
			Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(resp.Msg)
	}
	return resp.Data.ImageKey, nil
}

func (a *ActionInfo) downloadFile(fileKey string, fileName string, msgId *string) error {
	req := larkim.NewGetMessageResourceReqBuilder().MessageId(*msgId).FileKey(fileKey).Type("file").Build()
	resp, err := a.larkClient.Im.MessageResource.Get(context.Background(), req)
	if err != nil {
		return err
	}

	resp.WriteFile(fileName)
	return nil
}

func (a *ActionInfo) uploadOpus(f *os.File, fileName string) (string, error) {
	audioReq := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType("opus").
			FileName(fileName).
			File(f).
			Build()).
		Build()
	client := a.larkClient
	resp, err := client.Im.File.Create(context.Background(), audioReq)
	// 处理错误
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return "", errors.New(resp.Msg)
	}
	return *resp.Data.FileKey, nil
}

func (a *ActionInfo) replyImage(ctx context.Context, ImageKey *string,
	msgId *string) error {
	//fmt.Println("sendMsg", ImageKey, msgId)

	msgImage := larkim.MessageImage{ImageKey: *ImageKey}
	content, err := msgImage.String()
	if err != nil {
		fmt.Println(err)
		return err
	}
	client := a.larkClient

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeImage).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func (a *ActionInfo) UpdateImageCard(ctx context.Context, base64Str string, msgId *string, sessionId *string, question string) error {
	imageKey, err := a.uploadImage(base64Str)
	if err != nil {
		return err
	}
	var newCard string

	newCard, _ = newSendCard(
		withHeader(" ", larkcard.TemplateBlue),
		withImg(*imageKey, question),
		withNote("已完成，您可以继续提问或者选择其他功能。"))

	err = a.PatchCard(ctx, msgId, newCard)
	if err != nil {
		return err
	}
	return nil
}

func (a *ActionInfo) replayImageCardByBase64(ctx context.Context, base64Str string, msgId *string, sessionId *string, question string) error {
	imageKey, err := a.uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = a.sendImageCard(ctx, *imageKey, msgId, sessionId, question)
	if err != nil {
		return err
	}
	return nil
}

func (a *ActionInfo) replayImagePlainByBase64(ctx context.Context, base64Str string,
	msgId *string) error {
	imageKey, err := a.uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = a.replyImage(ctx, imageKey, msgId)
	if err != nil {
		return err
	}
	return nil
}

func (a *ActionInfo) replayVariantImageByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string) error {
	imageKey, err := a.uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = a.sendVarImageCard(ctx, *imageKey, msgId, sessionId)
	if err != nil {
		return err
	}
	return nil
}

func (a *ActionInfo) sendMsg(ctx context.Context, msg string, chatId *string) error {
	//fmt.Println("sendMsg", msg, chatId)
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := a.larkClient
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	//fmt.Println("content", content)

	resp, err := client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId(*chatId).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func (a *ActionInfo) alert(ctx context.Context, msg string) error {
	//fmt.Println("sendMsg", msg, chatId)
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := a.larkClient
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	//fmt.Println("content", content)
	// oc_ab7d028d1163c0575fbc0cc38e1e66e6   qaq
	// oc_9f663609fa6912bad3a37ef97f58fdeb yh
	resp, err := client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId("oc_9f663609fa6912bad3a37ef97f58fdeb").
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return err
	}
	return nil
}

func (a *ActionInfo) sendSystemInstructionCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🥷  已进入角色扮演模式", larkcard.TemplateIndigo),
		withMainText(content),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"))
	a.replyCard(ctx, msgId, newCard)
}

func (a *ActionInfo) sendNewTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("👻️ 已开启新的话题", larkcard.TemplateBlue),
		withMainMd(content),
		withNote("提醒：点击对话框参与回复，可保持话题连贯"))
	a.replyCard(ctx, msgId, newCard)
}

func (a *ActionInfo) sendOldTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
		withMainMd(content),
		withNote("提醒：点击对话框参与回复，可保持话题连贯"))
	a.replyCard(ctx, msgId, newCard)
}

func (a *ActionInfo) sendHelpCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("需要帮助吗？", larkcard.TemplateBlue),
		withMainMd("直接输入文字聊天"),
		withMainMd("直接发送文件用于上传文件"),
		withSplitLine(),
		withMainMd("/files 获取所有已上传的文件"),
		withMainMd("/delete *id* 删除id对应的文件"),
		withMainMd("/preview *id* 预览id对应的文件内容"),
		withMainMd("/read *id* *prompt* 基于id对应的文件进行对话"),
	)
	a.replyCard(ctx, msgId, newCard)
}

func (a *ActionInfo) sendImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string, question string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
	)
	a.replyCard(ctx, msgId, newCard)
	return nil
}

func (a *ActionInfo) sendVarImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn("再来一张", map[string]interface{}{
			"value":     imageKey,
			"kind":      PicVarMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	a.replyCard(ctx, msgId, newCard)
	return nil
}

func (a *ActionInfo) SendRoleTagsCard(ctx context.Context,
	sessionId *string, msgId *string, roleTags []string) {
	newCard, _ := newSendCard(
		withHeader("🛖 请选择角色类别", larkcard.TemplateIndigo),
		withRoleTagsBtn(sessionId, roleTags...),
		withNote("提醒：选择角色所属分类，以便我们为您推荐更多相关角色。"))
	err := a.replyCard(ctx, msgId, newCard)
	if err != nil {
		log.Printf("选择角色出错 %v", err)
	}
}

func (a *ActionInfo) SendRoleListCard(ctx context.Context,
	sessionId *string, msgId *string, roleTag string, roleList []string) {
	newCard, _ := newSendCard(
		withHeader("🛖 角色列表"+" - "+roleTag, larkcard.TemplateIndigo),
		withRoleBtn(sessionId, roleList...),
		withNote("提醒：选择内置场景，快速进入角色扮演模式。"))
	a.replyCard(ctx, msgId, newCard)
}

func (a *ActionInfo) SendAIModeListsCard(ctx context.Context,
	sessionId *string, msgId *string, aiModeStrs []string) {
	newCard, _ := newSendCard(
		withHeader("🤖 发散模式选择", larkcard.TemplateIndigo),
		withAIModeBtn(sessionId, aiModeStrs),
		withNote("提醒：选择内置模式，让AI更好的理解您的需求。"))
	a.replyCard(ctx, msgId, newCard)
}

func (a *ActionInfo) sendOnProcessCard(ctx context.Context,
	sessionId *string, msgId *string, ifNewTopic bool) (*string,
	error) {
	var newCard string
	if ifNewTopic {
		newCard, _ = newSendCard(
			withHeader("👻️ 已开启新的话题", larkcard.TemplateBlue),
			withNote("正在思考，请稍等..."))
	} else {
		newCard, _ = newSendCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
			withNote("正在思考，请稍等..."))
	}

	id, err := a.replyCardWithBackId(ctx, msgId, newCard)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (a *ActionInfo) UpdateTextCard(ctx context.Context, msg string, msgId *string, ifNewTopic bool) error {
	var newCard string
	if ifNewTopic {
		newCard, _ = newSendCard(
			withHeader("👻️ 已开启新的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withNote("正在生成，请稍等..."))
	} else {
		newCard, _ = newSendCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withNote("正在生成，请稍等..."))
	}
	err := a.PatchCard(ctx, msgId, newCard)
	if err != nil {
		return err
	}
	return nil
}
func (a *ActionInfo) updateFinalCard(
	ctx context.Context,
	msg string,
	msgId *string,
	ifNewSession bool,
) error {
	var newCard string
	if ifNewSession {
		newCard, _ = newSendCard(
			withHeader("👻️ 已开启新的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withNote("已完成，您可以继续提问或者选择其他功能。"))
	} else {
		newCard, _ = newSendCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),

			withMainMd(msg),
			withNote("已完成，您可以继续提问或者选择其他功能。"))
	}
	err := a.PatchCard(ctx, msgId, newCard)
	if err != nil {
		return err
	}
	return nil
}

func newSendCardWithOutHeader(
	elements ...larkcard.MessageCardElement) (string, error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(true).
		Build()
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

func (a *ActionInfo) PatchCard(ctx context.Context, msgId *string,
	cardContent string) error {
	//fmt.Println("sendMsg", msg, chatId)
	client := a.larkClient
	//content := larkim.NewTextMsgBuilder().
	//	Text(msg).
	//	Build()

	//fmt.Println("content", content)

	resp, err := client.Im.Message.Patch(ctx, larkim.NewPatchMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewPatchMessageReqBodyBuilder().
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func (a *ActionInfo) replyCardWithBackId(ctx context.Context,
	msgId *string,
	cardContent string,
) (*string, error) {
	client := a.larkClient
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Uuid(uuid.New().String()).
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(resp.Msg)
	}

	//ctx = context.WithValue(ctx, "SendMsgId", *resp.Data.MessageId)
	//SendMsgId := ctx.Value("SendMsgId")
	//pp.Println(SendMsgId)
	return resp.Data.MessageId, nil
}
