#include <metal_stdlib>
using namespace metal;

struct Vertex {
    float2 position;
};

struct Uniforms {
    float2 center;
    float zoom;
    float time;
    float2 fractalConstant;
    float2 resolution;
    int maxIterations;
    int pad0;
    int pad1;
    int pad2;
};

static float3 palette(float t) {
    const float3 a = float3(0.5, 0.5, 0.5);
    const float3 b = float3(0.5, 0.5, 0.5);
    const float3 c = float3(1.0, 1.0, 1.0);
    const float3 d = float3(0.0, 0.33, 0.67);
    return a + b * cos(6.28318 * (c * t + d));
}

static float3 mandelbrot_color_at(float2 fragCoord, constant Uniforms &u) {
    float scale = min(u.resolution.x, u.resolution.y);
    float2 p = (fragCoord - u.resolution * 0.5) / scale;
    float2 c = u.center + p * u.zoom;
    float2 z = float2(0.0, 0.0);

    float trap = 1.0e9;

    int i = 0;
    for (; i < u.maxIterations; i++) {
        float len2 = dot(z, z);
        trap = min(trap, sqrt(len2));

        if (len2 > 256.0) {
            break;
        }
        if (len2 > 4.0) {
            break;
        }

        z = float2(z.x * z.x - z.y * z.y, 2.0 * z.x * z.y) + c;
    }

    if (i >= u.maxIterations) {
        float interiorPhase = 0.12 + u.time * 0.004 + 0.08 * sin((p.x + p.y) * 7.0);
        float3 interior = palette(interiorPhase);
        // Keep interiors subtle but visibly non-black so they do not look like a blank screen.
        return interior * 0.26 + float3(0.070, 0.070, 0.085);
    }

    float mag = max(length(z), 1.0001);
    float mu = float(i) - log2(log2(mag));
    float depthPhase = log(max(1.0 / max(u.zoom, 1.0e-6), 1.0));
    float3 color = palette(mu * 0.05 + u.time * 0.008 + depthPhase * 0.2);
    color *= 0.6 + 0.4 * exp(-trap * 5.0);
    return color;
}

vertex float4 vertex_main(
    const device Vertex *vertices [[buffer(0)]],
    uint vid [[vertex_id]]
) {
    return float4(vertices[vid].position, 0.0, 1.0);
}

fragment float4 fragment_main(
    float4 position [[position]],
    constant Uniforms &u [[buffer(1)]]
) {
    constexpr bool kDebugGradient = false;
    if (kDebugGradient) {
        return float4(
            position.x / max(u.resolution.x, 1.0),
            position.y / max(u.resolution.y, 1.0),
            0.5,
            1.0
        );
    }

    float2 base = position.xy;
    float3 c = float3(0.0);
    c += mandelbrot_color_at(base + float2(0.25, 0.25), u);
    c += mandelbrot_color_at(base + float2(-0.25, 0.25), u);
    c += mandelbrot_color_at(base + float2(0.25, -0.25), u);
    c += mandelbrot_color_at(base + float2(-0.25, -0.25), u);
    c *= 0.25;

    return float4(c, 1.0);
}
