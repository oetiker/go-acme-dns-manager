package manager

// ConfigSchema defines the JSON schema for the configuration file
const ConfigSchema = `{
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"title": "go-acme-dns-manager configuration",
	"type": "object",
	"required": ["email", "acme_server", "acme_dns_server"],
	"additionalProperties": false,
	"properties": {
		"email": {
			"type": "string",
			"format": "email",
			"description": "Email address for Let's Encrypt registration and notifications"
		},
		"acme_server": {
			"type": "string",
			"format": "uri",
			"description": "Let's Encrypt ACME server URL"
		},
		"acme_dns_server": {
			"type": "string",
			"format": "uri",
			"description": "URL of your acme-dns server"
		},
		"key_type": {
			"type": "string",
			"enum": ["rsa2048", "rsa3072", "rsa4096", "ec256", "ec384"],
			"description": "Key type for the certificate"
		},
		"dns_resolver": {
			"type": "string",
			"description": "DNS resolver to use for CNAME verification checks"
		},
		"cert_storage_path": {
			"type": "string",
			"description": "Path where Let's Encrypt certificates, account info, and acme-dns credentials will be stored"
		},
		"challenge_timeout": {
			"type": "string",
			"description": "Timeout for ACME challenges (e.g., DNS propagation checks). Format: Go duration string"
		},
		"http_timeout": {
			"type": "string",
			"description": "Timeout for HTTP requests made to the ACME server. Format: Go duration string"
		},
		"auto_domains": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
				"grace_days": {
					"type": "integer",
					"minimum": 1,
					"description": "Renew certs expiring within this many days"
				},
				"certs": {
					"type": "object",
					"additionalProperties": {
						"type": "object",
						"required": ["domains"],
						"additionalProperties": false,
						"properties": {
							"key_type": {
								"type": "string",
								"enum": ["rsa2048", "rsa3072", "rsa4096", "ec256", "ec384"],
								"description": "Override global key_type for this cert"
							},
							"domains": {
								"type": "array",
								"items": {
									"type": "string"
								},
								"minItems": 1,
								"description": "List of domains to include in the certificate"
							}
						}
					}
				}
			}
		}
	}
}`
