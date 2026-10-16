if defined?(OpenAI::Client)
  module OpenAIJsonObjectHint
    def chat(parameters: {})
      super(parameters: OpenAIJsonObjectHint.with_json_hint(parameters))
    end

    def self.with_json_hint(parameters)
      format = parameters[:response_format] || parameters["response_format"]
      return parameters unless format.is_a?(Hash) && (format[:type] || format["type"]).to_s == "json_object"

      messages_key = parameters.key?(:messages) ? :messages : "messages"
      messages = parameters[messages_key]
      return parameters unless messages.is_a?(Array)
      return parameters if messages.any? { |m| m.is_a?(Hash) && (m[:role] || m["role"]).to_s != "system" && (m[:content] || m["content"]).to_s.match?(/json/i) }

      index = messages.rindex { |m| m.is_a?(Hash) && (m[:role] || m["role"]).to_s == "user" && (m[:content] || m["content"]).is_a?(String) }
      return parameters unless index

      message = messages[index]
      content_key = message.key?(:content) ? :content : "content"
      hinted = messages.dup
      hinted[index] = message.merge(content_key => "Respond in JSON.\n\n#{message[content_key]}")
      parameters.merge(messages_key => hinted)
    end
  end

  OpenAI::Client.prepend(OpenAIJsonObjectHint)
end
