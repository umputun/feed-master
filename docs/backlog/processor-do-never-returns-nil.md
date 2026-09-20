---
worth: maybe
where: app/proc/processor.go:Do
added: 2026-09-19
---
# Processor.Do never returns nil, so the error log at main.go:93 is unreachable

`Processor.Do` is an infinite `for`/`select` whose only `return` is
`fmt.Errorf("processor stopped: %w", ctx.Err())` on `ctx.Done()`. It cannot return `nil`. `app/main.go:93`
calls it as `if err := p.Do(context.Background()); err != nil`, and `context.Background()` never cancels,
so `Do` never returns at all and the `[ERROR] processor failed` log below it is dead code.

golangci-lint v2.13 reports this as `SA4023: this comparison is always true` at `app/main.go:93:41`. The
CI pin is v2.12, which does not carry the check, so nothing fails today. It will surface on the next pin
bump.

`youtube.Service.Do` at `app/main.go:144` has the same shape, called with `context.TODO()`.

Three ways out, and nobody has ruled between them, which is why this is `maybe` rather than `later`:
give `main` a cancellable context so shutdown actually reaches these loops and the error log does
something; change `Do` to return `nil` on a clean stop; or drop the dead branch and let `Do` not return
an error at all. The first is the only one that adds behaviour rather than removing a symptom, and it is
also the largest.

Surfaced by a revmux round on the golangci-lint v2.12 branch (#174), not by that change.
