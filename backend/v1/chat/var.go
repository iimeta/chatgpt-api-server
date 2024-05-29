package chat

var (
	// client    = g.Client()
	ErrNoAuth = `{
		"error": {
			"message": "You didn't provide an API key. You need to provide your API key in an Authorization header using Bearer auth (i.e. Authorization: Bearer YOUR_KEY), or as the password field (with blank username) if you're accessing the API from your browser and are prompted for a username and password. You can obtain an API key from https://platform.openai.com/account/api-keys.",
			"type": "invalid_request_error",
			"param": null,
			"code": null
		}
	}`
	ErrKeyInvalid = `{
		"error": {
			"message": "Incorrect API key provided: sk-4yNZz***************************************6mjw. You can find your API key at https://platform.openai.com/account/api-keys.",
			"type": "invalid_request_error",
			"param": null,
			"code": "invalid_api_key"
		}
	}`
	ChatReqStr = `{
		"action": "next",
		"messages": [
		  {
			"id": "aaa2f210-64e1-4f0d-aa51-e73fe1ae74af",
			"author": { "role": "user" },
			"content": { "content_type": "text", "parts": ["1\n"] },
			"metadata": {}
		  }
		],
		"parent_message_id": "aaa1a8ab-61d6-4fc0-a5f5-181015c2ebaf",
		"model": "text-davinci-002-render-sha",
		"timezone_offset_min": -480,
		"suggestions": [],
		"history_and_training_disabled": true,
		"conversation_mode": { "kind": "primary_assistant" }
	  }
	  `
	ChatTurboReqStr = `
	{
		"action": "next",
		"messages": [
			{
				"id": "aaa2b2cc-e7e9-47c5-8171-0ff8a6d9d6d3",
				"author": {
					"role": "user"
				},
				"content": {
					"content_type": "text",
					"parts": [
						"你好"
					]
				},
				"metadata": {}
			}
		],
		"parent_message_id": "aaa1403d-c61e-4818-90e0-93a99465aec6",
		"model": "gpt-4",
		"timezone_offset_min": -480,
		"suggestions": [],
		"history_and_training_disabled": true,
		"conversation_mode": {
			"kind": "primary_assistant"
		}
	}`
	Chat4ReqStr = `
	{
		"action": "next",
		"messages": [
			{
				"id": "aaa2b182-d834-4f30-91f3-f791fa953204",
				"author": {
					"role": "user"
				},
				"content": {
					"content_type": "text",
					"parts": [
						"画一只猫1231231231"
					]
				},
				"metadata": {}
			}
		],
		"parent_message_id": "aaa11581-bceb-46c5-bc76-cb84be69725e",
		"model": "gpt-4-gizmo",
		"timezone_offset_min": -480,
		"suggestions": [],
		"history_and_training_disabled": true,
		"conversation_mode": {
			"gizmo": {
				"gizmo": {
					"id": "g-YyyyMT9XH",
					"organization_id": "org-OROoM5KiDq6bcfid37dQx4z4",
					"short_url": "g-YyyyMT9XH-chatgpt-classic",
					"author": {
						"user_id": "user-u7SVk5APwT622QC7DPe41GHJ",
						"display_name": "ChatGPT",
						"selected_display": "name",
						"is_verified": true
					},
					"voice": {
						"id": "ember"
					},
					"display": {
						"name": "ChatGPT Classic",
						"description": "The latest version of GPT-4 with no additional capabilities",
						"welcome_message": "Hello",
						"profile_picture_url": "https://files.oaiusercontent.com/file-i9IUxiJyRubSIOooY5XyfcmP?se=2123-10-13T01%3A11%3A31Z&sp=r&sv=2021-08-06&sr=b&rscc=max-age%3D31536000%2C%20immutable&rscd=attachment%3B%20filename%3Dgpt-4.jpg&sig=ZZP%2B7IWlgVpHrIdhD1C9wZqIvEPkTLfMIjx4PFezhfE%3D",
						"categories": []
					},
					"share_recipient": "link",
					"updated_at": "2023-11-06T01:11:32.191060+00:00",
					"last_interacted_at": "2023-11-18T07:50:19.340421+00:00",
					"tags": [
						"public",
						"first_party"
					]
				},
				"tools": [],
				"files": [],
				"product_features": {
					"attachments": {
						"type": "retrieval",
						"accepted_mime_types": [
							"text/x-c",
							"text/html",
							"application/x-latext",
							"text/plain",
							"text/x-ruby",
							"text/x-typescript",
							"text/x-c++",
							"text/x-java",
							"text/x-sh",
							"application/vnd.openxmlformats-officedocument.presentationml.presentation",
							"text/x-script.python",
							"text/javascript",
							"text/x-tex",
							"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
							"application/msword",
							"application/pdf",
							"text/x-php",
							"text/markdown",
							"application/json",
							"text/x-csharp"
						],
						"image_mime_types": [
							"image/jpeg",
							"image/png",
							"image/gif",
							"image/webp"
						],
						"can_accept_all_mime_types": true
					}
				}
			},
			"kind": "gizmo_interaction",
			"gizmo_id": "g-YyyyMT9XH"
		},
		"force_paragen": false,
		"force_rate_limit": false
	}`
	ApiRespStr = `{
		"id": "chatcmpl-LLKfuOEHqVW2AtHks7wAekyrnPAoj",
		"object": "chat.completion",
		"created": 1689864805,
		"model": "gpt-3.5-turbo",
		"usage": {
			"prompt_tokens": 0,
			"completion_tokens": 0,
			"total_tokens": 0
		},
		"choices": [
			{
				"message": {
					"role": "assistant",
					"content": "Hello! How can I assist you today?"
				},
				"finish_reason": "stop",
				"index": 0
			}
		]
	}`
	ApiRespStrStream = `{
		"id": "chatcmpl-afUFyvbTa7259yNeDqaHRBQxH2PLH",
		"object": "chat.completion.chunk",
		"created": 1689867370,
		"model": "gpt-3.5-turbo",
		"choices": [
			{
				"delta": {
					"content": "Hello"
				},
				"index": 0,
				"finish_reason": null
			}
		]
	}`
	ApiRespStrStreamEnd = `{"id":"apirespid","object":"chat.completion.chunk","created":apicreated,"model": "apirespmodel","choices":[{"delta": {},"index": 0,"finish_reason": "stop"}]}`
)
