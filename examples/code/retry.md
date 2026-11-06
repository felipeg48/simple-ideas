---
type: article
title: The retry, step by step
description: How the checkout's send() grew from one attempt to a retry that waits, spreads out and gives up on the right errors, one change at a time.
author: Jonas Berg · Payments team
date: 6 November 2026
theme: studio
footer: Northwind Logistics engineering
---

# The retry, step by step

The function that sends an order to the payment provider is twelve lines
long today. It took four changes to get there, and each one fixed a night we
remember.

```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers transition=morph step="One attempt"
func send(order Order) error {
	resp, err := client.Post(url, order)
	if err != nil {
		return err
	}
	return check(resp)
}
```
<!-- callout: 2 One attempt: a dropped connection loses the order. at=60,-54 -->

```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers transition=morph step="Retry five times"
func send(order Order) error {
	var last error
	for try := 1; try <= 5; try++ {
		resp, err := client.Post(url, order)
		if err == nil {
			return check(resp)
		}
		last = err
	}
	return last
}
```
<!-- callout: 3 Five tries, back to back. at=60,-46 -->

```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers transition=morph step="Back off" hl=9
func send(order Order) error {
	var last error
	for try := 1; try <= 5; try++ {
		resp, err := client.Post(url, order)
		if err == nil {
			return check(resp)
		}
		last = err
		time.Sleep(backoff(try))
	}
	return last
}
```
<!-- callout: 9 Wait 0.2 s, 0.4 s, 0.8 s… before the next try. at=60,30 -->

```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers transition=morph step="Only what can recover" hl=8-10,12
func send(order Order) error {
	var last error
	for try := 1; try <= 5; try++ {
		resp, err := client.Post(url, order)
		if err == nil {
			return check(resp)
		}
		if !isNetwork(err) {
			return err // a decline is final
		}
		last = err
		time.Sleep(backoff(try) + jitter())
	}
	return last
}
```
<!-- callout: 8 Declines are final: never retry a refused card. at=70,-64 -->
<!-- callout: 12 Jitter spreads the retries of a thousand orders. at=50,40 -->

## The same rule in the .NET gateway

The courier app talks to payments through a small gateway written in C#. It
uses the same rule, through a retry policy instead of a loop.

```csharp window=windows title="PaymentClient.cs - Gateway" theme=paper backdrop=ocean
public sealed class PaymentClient
{
    private static readonly ResiliencePipeline Retry = new ResiliencePipelineBuilder()
        .AddRetry(new() { MaxRetryAttempts = 4, BackoffType = DelayBackoffType.Exponential, UseJitter = true })
        .Build();

    public Task<Receipt> SendAsync(Order order) =>
        Retry.ExecuteAsync(async ct => await _http.PostAsJsonAsync<Receipt>("/charge", order, ct)).AsTask();
}
```
<!-- callout: 4 Four retries, growing waits, with jitter. -->

```xml window=windows title="Gateway.csproj" theme=paper
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Polly.Core" Version="8.4.1" />
  </ItemGroup>
</Project>
```

## Proving it before Thursday

The test that guards the rule runs a hundred times before every release.

```sh window=terminal title="checkout" backdrop=graphite
go test ./payments -run TestRetryGivesUpOnDeclines -count=100
ok  	northwind/payments	3.214s
```
