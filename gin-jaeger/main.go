package main

import (
	"context"
	"errors"
	"sync"

	"github.com/gin-gonic/gin"
	uuid "github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	r := gin.New()
	// 初始化并配置 Tracer
	ctx := context.Background()
	tp, err := initTracer(ctx)
	if err != nil {
		log.Err(err).Msg("initTracer failed")
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Err(err).Msg("shutdown tracer provider failed")
			return
		}
	}()

	//中间件
	r.Use(otelgin.Middleware(serviceName))

	var tracer = otel.Tracer("gin-server")
	//创建一个 Span
	r.GET("/hello", func(c *gin.Context) {
		id, _ := uuid.NewV4()
		_, span := tracer.Start(
			c.Request.Context(), "hello", trace.WithAttributes(attribute.String("id", id.String())),
		)
		defer span.End()

	})

	//Span的父子关系
	r.GET("/childrenspan", func(c *gin.Context) {
		spanCtx, span := tracer.Start(c.Request.Context(), "parentSpan")
		defer span.End()
		func(ctx context.Context) {
			_, span := tracer.Start(spanCtx, "children", trace.WithAttributes(attribute.String("key", "value")))
			defer span.End()
		}(c.Request.Context())
	})

	//事件
	//event事件代表生命周期内发生的事，如在一个函数中的不同节点进行事件记录，如同log
	r.GET("/event", func(c *gin.Context) {
		_, span := tracer.Start(c.Request.Context(), "event")
		defer span.End()
		var mu sync.Mutex

		span.AddEvent("waiting lock", trace.WithAttributes(attribute.String("key", "value")))
		mu.Lock()
		span.AddEvent("locked lock", trace.WithAttributes(attribute.String("key", "value")))
		mu.Unlock()
		span.AddEvent("unlocked lock", trace.WithAttributes(attribute.Int("pid", 4328), attribute.String("key", "value")))
	})

	// 记录捕获错误
	r.GET("/error", func(c *gin.Context) {
		errExample := errors.New("error")
		_, span := tracer.Start(c.Request.Context(), "error")
		defer span.End()
		span.SetStatus(codes.Error, errExample.Error())
		span.RecordError(errExample)
	})

	if err := r.Run(":8080"); err != nil {
		log.Err(err).Msg("gin run failed")
		return
	}

}
