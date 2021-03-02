package providers

import (
	"github.com/xxxmicro/framework/config"
	xconfig "github.com/xxxmicro/framework/config"
	xsource "github.com/xxxmicro/framework/config/source"
	"github.com/xxxmicro/framework/opentracing/jaeger"
	"go.uber.org/fx"
)

var Framework = fx.Provide(
	config.NewAppConfig,
	xsource.NewSourceProvider,
	xconfig.NewConfigProvider,
	jaeger.NewTracerProvider,
)

var FrameworkOpts = fx.Options(
	Framework,
	fx.Invoke(InitLogger),
)
