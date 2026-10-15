if defined?(Logtail::Integrations::Rack::HTTPEvents)
  Logtail::Integrations::Rack::HTTPEvents.http_header_filters = %w[Authorization Cookie Set-Cookie X-Api-Key]
end
