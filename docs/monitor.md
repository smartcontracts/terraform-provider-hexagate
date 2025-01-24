# hexagate_monitor Resource

Manages a Hexagate monitor.

## Example Usage

```tf
resource "hexagate_monitor" "example" {
  name        = "Example Balance Monitor"
  monitor_id  = 1
  description = "An example showing how to configure a balance monitor"
  disabled    = false

  entities {
    entity_type = 1
    params      = jsonencode({
      type     = 1
      address  = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
      chain_id = 1
    })
  }
  
  monitor_rules {
    name       = "Example Rule"
    type       = "notification"
    threshold  = 30
    categories = [1, 2, 3]

    channels {
      id                  = 1111
      name                = "Example Channel"
      event_types         = null
      notification_period = null
      params              = jsonencode({
        include_all       = true
        channel           = null
        username          = null
        fields_to_include = []
        type              = 1
        identity          = "https://example.com/webhook"
      })
    }
  }
  
  params = jsonencode({
    type = 4
    severity = 30
    monitor_conditions {
      is_usd_value = false
      token        = null
      condition    = {
        amount = 1
        type   = 4
      }

      tokens {
        address  = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
        chain_id = 1
      }
    }
  })
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the monitor
* `monitor_id` - (Optional) The ID of the monitor type
* `description` - (Optional) A description of the monitor
* `disabled` - (Required) Whether the monitor is disabled
* `entities` - (Optional) A list of entities to monitor. Each entity block supports:
  * `entity_type` - (Required) The type of the entity
  * `params` - (Required) JSON encoded parameters for the entity
* `monitor_rules` - (Optional) A list of rules for the monitor. Each rule block supports:
  * `name` - (Required) The name of the rule
  * `type` - (Required) The type of the rule
  * `threshold` - (Required) The threshold for the rule
  * `categories` - (Required) List of category IDs
  * `channels` - (Optional) List of notification channels. Each channel block supports:
    * `name` - (Required) The name of the channel
    * `params` - (Required) JSON encoded parameters for the channel
* `params` - (Optional) JSON encoded parameters for the monitor

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the monitor
* `created_by` - The creator of the monitor
* `created_at` - The creation timestamp
* `updated_at` - The last update timestamp

## Import

Monitors can be imported using their ID:

```sh
terraform import hexagate_monitor.example 12345
```
