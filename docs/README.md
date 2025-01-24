# Hexagate Provider

The Hexagate Provider enables Terraform to manage [Hexagate](https://www.hexagate.com/) resources.

## Example Usage

```tf
terraform {
  required_providers {
    hexagate = {
      source = "smartcontracts/hexagate"
    }
  }
}

provider "hexagate" {
  api_token = "your-api-token"
  api_url   = "https://api.hexagate.com/api/v2"  # Optional
}
```

## Authentication

The Hexagate provider requires an API token for authentication. This can be provided in the provider configuration block.

## Provider Arguments

* `api_token` (Required) - Hexagate API token for authentication
* `api_url` (Optional) - The URL of the Hexagate API. Defaults to `https://api.hexagate.com/api/v2`

## Resources

* [hexagate_monitor](./monitor.md)
