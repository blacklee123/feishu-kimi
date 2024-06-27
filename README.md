# feishu-kimi

<p align='center'>
   飞书 × kimi 
<br>
<br>
    🚀 Feishu Kimi 🚀
</p>

## 👻 机器人功能
1. 文字聊天
2. 基于文件的文字聊天

## 🌟 项目特点

- 🍏 对话基于 Kimi(https://platform.moonshot.cn/docs/) 接口
- 🍎 通过 lark，将 Kimi 接入[飞书](https://open.feishu.cn/app)
- 基于飞书长连接事件回调，不需要公网IP
- 基于飞书消息更新，流式回复

## 项目部署

### OpenAI部署

```bash
docker run -d --restart=always --name feishu-kimi \
--env FEISHUAPP_ID=xxx \
--env FEISHUAPP_SECRET=xxx \
--env FEISHU_ENCRYPT_KEY=xxx \
--env FEISHU_VERIFICATION_TOKEN=xxx \
--env OPENAI_MODEL=moonshot-v1-128k \
--env OPENAI_API_URL=https://api.moonshot.cn/v1 \
--env OPENAI_KEY=sk-xxx1 \
blacklee123/feishu-kimi:latest
```

## 详细配置步骤



- 获取 [Kimi](https://platform.moonshot.cn/console/api-keys) 的 KEY
- 创建 [飞书](https://open.feishu.cn/) 机器人
    1. 前往[开发者平台](https://open.feishu.cn/app?lang=zh-CN)创建应用,并获取到 APPID 和 Secret
    2. 前往`添加应用能力`, 添加机器人
    3. 进入`权限管理`界面。添加下列权限
        - contact:contact.base:readonly(获取通讯录基本信息)
        - contact:user.base:readonly(获取用户基本信息)
        - im:resource(获取与上传图片或文件资源)
        - im:message
        - im:message.group_at_msg:readonly(接收群聊中@机器人消息事件)
        - im:message.p2p_msg(获取用户发给机器人的单聊消息)
        - im:message.p2p_msg:readonly(读取用户发给机器人的单聊消息)
        - im:message:send_as_bot(获取用户在群组中@机器人的消息)
    4. 进入`事件与回调-事件配置` 
        1. 配置订阅方式为`使用长链接接收事件`
        2. 添加事件，接收消息im.message.receive_v1
    5. 发布版本，等待企业管理员审核通过

## 加入答疑群

[单击加入答疑群](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=1e9haaa0-1260-44b3-8286-f8cc926fa385)
