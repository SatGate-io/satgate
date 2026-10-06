'use client';

import React, { useEffect, useRef, useState } from 'react';
import { Pause, Play, Volume2, VolumeX } from 'lucide-react';

// The overview commercial loops silently at the top of the homepage; its captions carry the story.
// "Sound on" restarts it from the beginning with the voice, plays it once, then it returns to the silent loop.
// It doesn't autoplay for visitors who prefer reduced motion, and the pause button can stop it at any time
// (content that moves for more than five seconds needs a way to pause).
// Play state comes only from the video's own play/pause events.
export default function HeroVideo() {
  const ref = useRef<HTMLVideoElement>(null);
  const [playing, setPlaying] = useState(false);
  const [muted, setMuted] = useState(true);

  useEffect(() => {
    const v = ref.current;
    if (!v) return;
    v.muted = true;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    v.play().catch(() => {
      // Autoplay blocked (e.g. low-power mode): the poster and Play button stay up.
    });
  }, []);

  const play = () => {
    ref.current?.play().catch(() => {});
  };

  const togglePlay = () => {
    const v = ref.current;
    if (!v) return;
    if (v.paused) play();
    else v.pause();
  };

  const toggleSound = () => {
    const v = ref.current;
    if (!v) return;
    if (v.muted) {
      v.muted = false;
      v.loop = false;
      v.currentTime = 0;
      setMuted(false);
      play();
    } else {
      v.muted = true;
      v.loop = true;
      setMuted(true);
    }
  };

  const backToSilentLoop = () => {
    const v = ref.current;
    if (!v) return;
    v.muted = true;
    v.loop = true;
    setMuted(true);
    v.currentTime = 0;
    play();
  };

  const control = 'flex items-center gap-1.5 rounded-md bg-gray-800/80 px-2.5 py-1.5 text-xs text-gray-300 transition hover:bg-gray-700 hover:text-white';

  return (
    <div className="relative bg-gray-900 rounded-xl border border-gray-800 overflow-hidden shadow-2xl">
      <div className="flex items-center gap-2 px-4 py-2.5 border-b border-gray-800 bg-gray-900/80">
        <div className="w-3 h-3 rounded-full bg-red-500"></div>
        <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
        <div className="w-3 h-3 rounded-full bg-green-500"></div>
        <div className="hidden sm:block text-xs text-gray-500 ml-2 font-mono">SatGate overview</div>
        <div className="ml-auto flex items-center gap-2">
          <button type="button" onClick={toggleSound} className={control}>
            {muted ? <VolumeX size={14} /> : <Volume2 size={14} />}
            {muted ? 'Sound on' : 'Mute'}
          </button>
          <button type="button" onClick={togglePlay} aria-label={playing ? 'Pause the video' : 'Play the video'} className={control}>
            {playing ? <Pause size={14} /> : <Play size={14} />}
          </button>
        </div>
      </div>
      <div className="relative">
        <video
          ref={ref}
          loop
          muted
          playsInline
          preload="metadata"
          poster="/satgate-overview-poster.jpg"
          className="block w-full"
          aria-label="SatGate overview with captions: an agent stuck in a loop at 3 AM, a meter for agent spend, a budget the agent can't go past, a front door that charges external agents, and signed receipts anyone can check"
          onPlay={() => setPlaying(true)}
          onPause={() => setPlaying(false)}
          onEnded={backToSilentLoop}
        >
          <source src="/satgate-overview.mp4" type="video/mp4" />
        </video>
        {!playing && (
          <button
            type="button"
            onClick={play}
            aria-label="Play the overview video"
            className="absolute inset-0 flex items-center justify-center bg-black/30 transition hover:bg-black/20"
          >
            <span className="flex items-center gap-2 rounded-full bg-white/90 px-5 py-2.5 text-sm font-bold text-black shadow-lg">
              <Play size={16} /> Play
            </span>
          </button>
        )}
      </div>
    </div>
  );
}
