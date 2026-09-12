<div align="center">
    <h1>Terraform Provider for CTFd</h1>
    <p><b>Time for CTF(d) as Code</b><p>
    <a href="https://goreportcard.com/report/github.com/eliabir/terraform-provider-ctfd"><img src="https://goreportcard.com/badge/github.com/eliabir/terraform-provider-ctfd?style=for-the-badge" alt="go report"></a>
	<a href="https://github.com/eliabir/terraform-provider-ctfd/blob/main/LICENSE"><img src="https://img.shields.io/github/license/eliabir/terraform-provider-ctfd?style=for-the-badge" alt="License"></a>
	<br>
	<a href="https://github.com/eliabir/terraform-provider-ctfd/actions?query=workflow%3Aci+"><img src="https://img.shields.io/github/actions/workflow/status/eliabir/terraform-provider-ctfd/ci.yaml?style=for-the-badge&label=CI" alt="CI"></a>
	<a href="https://github.com/eliabir/terraform-provider-ctfd/actions/workflows/codeql-analysis.yaml"><img src="https://img.shields.io/github/actions/workflow/status/eliabir/terraform-provider-ctfd/codeql-analysis.yaml?style=for-the-badge&label=CodeQL" alt="CodeQL"></a>
</div>

> [!NOTE]
> This is a modified fork of
> [ctfer-io/terraform-provider-ctfd](https://github.com/ctfer-io/terraform-provider-ctfd).

## Why creating this ?

Terraform is used to manage resources that have lifecycles, configurations, to sum it up.

That is the case of CTFd: it handles challenges that could be created, modified and deleted.
With some work to leverage the unsteady CTFd's API, Terraform is now able to manage them as cloud resources bringing you to opportunity of **CTF as Code**.

With a paradigm-shifting vision of setting up CTFs, the Terraform Provider for CTFd avoid shitty scripts, `ctfcli` and other tools that does not solve the problem of reproductibility, ease of deployment and resiliency.

## How to use it ?

Install the **Terraform Provider for CTFd** by setting the following in your `main.tf file`.
```hcl
terraform {
    required_providers {
        ctfd = {
            source = "registry.opentofu.org/eliabir/ctfd"
        }
    }
}

provider "ctfd" {
    url = "https://my-ctfd.lan"
}
```

We recommend setting the environment variable `CTFD_API_KEY` to enable the provider to communicate with your CTFd instance.

Then, you could use a `ctfd_challenge_standard` resource to setup your CTFd challenges, with for instance the following configuration.
```hcl
resource "ctfd_challenge_standard" "my_challenge" {
    name        = "My Challenge"
    category    = "Some category"
    description = <<-EOT
        My superb description !

        And it's multiline :o
    EOT
    state       = "visible"
    value       = 500
}
```

## OpenTelemetry support

Understanding what is going on under the hood or what could fail throughout the CTF lifecycle remains an important concern, even with such provider. For better understandability, we ship support for OpenTelemetry.

You can configure it using [the SDK environment variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/).

Note that CTFd **does not support it natively**, you may want to use our [instrumented and packaged CTFd](https://github.com/ctfer-io/ctfd-packaged) or proceed similarly for auto-instrumentation.

Also, the provider uses the `always` sampler hence we recommend you use a [Collector probability sampler](https://opentelemetry.io/docs/specs/otel/trace/tracestate-probability-sampling/). An example follows, with arbitrary values.
```yaml
processors:
  probabilistic_sampler:
    hash_seed: 22
    sampling_percentage: 22

service:
  pipelines:
    traces:
      receivers: [...]
      processors: [probabilistic_sampler, ...]
      exporters: [...]
```

A more complete example is [available here](./examples/opentelemetry).
