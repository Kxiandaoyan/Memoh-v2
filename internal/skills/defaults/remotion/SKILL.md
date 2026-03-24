---
name: remotion-best-practices
description: "Create video compositions, render frames, add animations, configure audio tracks, sequence scenes, and export MP4 videos using Remotion in React. Use when working with Remotion, programmatic video generation, React video rendering, or video export pipelines."
metadata:
  tags: remotion, video, react, animation, composition
---

## Quick start

A minimal Remotion composition that renders a fade-in title:

```tsx
import { AbsoluteFill, interpolate, useCurrentFrame } from "remotion";

export const FadeInTitle: React.FC = () => {
  const frame = useCurrentFrame();
  const opacity = interpolate(frame, [0, 30], [0, 1], {
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ justifyContent: "center", alignItems: "center" }}>
      <h1 style={{ fontSize: 80, opacity }}>Hello Remotion</h1>
    </AbsoluteFill>
  );
};
```

Register it as a composition in `Root.tsx`:

```tsx
import { Composition } from "remotion";
import { FadeInTitle } from "./FadeInTitle";

export const RemotionRoot: React.FC = () => (
  <Composition id="FadeIn" component={FadeInTitle} durationInFrames={90} fps={30} width={1920} height={1080} />
);
```

## When to use

Use this skill whenever working with Remotion code — composing scenes, animating elements, embedding media, rendering video, or configuring audio and transitions.

## Start here

If you are new to this skill, read these rules first:

1. [rules/compositions.md](rules/compositions.md) — How to define compositions and set props
2. [rules/animations.md](rules/animations.md) — Core animation patterns (interpolate, spring)
3. [rules/sequencing.md](rules/sequencing.md) — Ordering and timing scenes
4. [rules/assets.md](rules/assets.md) — Importing images, videos, audio, and fonts

## Common workflows

**Media & assets**: [assets](rules/assets.md) · [images](rules/images.md) · [videos](rules/videos.md) · [audio](rules/audio.md) · [fonts](rules/fonts.md) · [gifs](rules/gifs.md)

**Animation & timing**: [animations](rules/animations.md) · [timing](rules/timing.md) · [sequencing](rules/sequencing.md) · [transitions](rules/transitions.md) · [text-animations](rules/text-animations.md) · [trimming](rules/trimming.md)

**Advanced**: [3d](rules/3d.md) · [lottie](rules/lottie.md) · [charts](rules/charts.md) · [maps](rules/maps.md) · [light-leaks](rules/light-leaks.md) · [transparent-videos](rules/transparent-videos.md)

**Configuration & utilities**: [compositions](rules/compositions.md) · [calculate-metadata](rules/calculate-metadata.md) · [parameters](rules/parameters.md) · [measuring-text](rules/measuring-text.md) · [measuring-dom-nodes](rules/measuring-dom-nodes.md) · [tailwind](rules/tailwind.md)

**Media inspection (Mediabunny)**: [can-decode](rules/can-decode.md) · [extract-frames](rules/extract-frames.md) · [get-audio-duration](rules/get-audio-duration.md) · [get-video-dimensions](rules/get-video-dimensions.md) · [get-video-duration](rules/get-video-duration.md)

## Captions

When dealing with captions or subtitles, load the [./rules/subtitles.md](./rules/subtitles.md) file for more information.

## All rules

Read individual rule files for detailed explanations and code examples:

- [rules/3d.md](rules/3d.md) - 3D content in Remotion using Three.js and React Three Fiber
- [rules/animations.md](rules/animations.md) - Fundamental animation skills for Remotion
- [rules/assets.md](rules/assets.md) - Importing images, videos, audio, and fonts into Remotion
- [rules/audio.md](rules/audio.md) - Using audio and sound in Remotion - importing, trimming, volume, speed, pitch
- [rules/calculate-metadata.md](rules/calculate-metadata.md) - Dynamically set composition duration, dimensions, and props
- [rules/can-decode.md](rules/can-decode.md) - Check if a video can be decoded by the browser using Mediabunny
- [rules/charts.md](rules/charts.md) - Chart and data visualization patterns for Remotion
- [rules/compositions.md](rules/compositions.md) - Defining compositions, stills, folders, default props and dynamic metadata
- [rules/extract-frames.md](rules/extract-frames.md) - Extract frames from videos at specific timestamps using Mediabunny
- [rules/fonts.md](rules/fonts.md) - Loading Google Fonts and local fonts in Remotion
- [rules/get-audio-duration.md](rules/get-audio-duration.md) - Getting the duration of an audio file in seconds with Mediabunny
- [rules/get-video-dimensions.md](rules/get-video-dimensions.md) - Getting the width and height of a video file with Mediabunny
- [rules/get-video-duration.md](rules/get-video-duration.md) - Getting the duration of a video file in seconds with Mediabunny
- [rules/gifs.md](rules/gifs.md) - Displaying GIFs synchronized with Remotion's timeline
- [rules/images.md](rules/images.md) - Embedding images in Remotion using the Img component
- [rules/light-leaks.md](rules/light-leaks.md) - Light leak overlay effects using @remotion/light-leaks
- [rules/lottie.md](rules/lottie.md) - Embedding Lottie animations in Remotion
- [rules/measuring-dom-nodes.md](rules/measuring-dom-nodes.md) - Measuring DOM element dimensions in Remotion
- [rules/measuring-text.md](rules/measuring-text.md) - Measuring text dimensions, fitting text to containers, and checking overflow
- [rules/sequencing.md](rules/sequencing.md) - Sequencing patterns for Remotion - delay, trim, limit duration of items
- [rules/tailwind.md](rules/tailwind.md) - Using TailwindCSS in Remotion
- [rules/text-animations.md](rules/text-animations.md) - Typography and text animation patterns for Remotion
- [rules/timing.md](rules/timing.md) - Interpolation curves in Remotion - linear, easing, spring animations
- [rules/transitions.md](rules/transitions.md) - Scene transition patterns for Remotion
- [rules/transparent-videos.md](rules/transparent-videos.md) - Rendering out a video with transparency
- [rules/trimming.md](rules/trimming.md) - Trimming patterns for Remotion - cut the beginning or end of animations
- [rules/videos.md](rules/videos.md) - Embedding videos in Remotion - trimming, volume, speed, looping, pitch
- [rules/parameters.md](rules/parameters.md) - Make a video parametrizable by adding a Zod schema
- [rules/maps.md](rules/maps.md) - Add a map using Mapbox and animate it
