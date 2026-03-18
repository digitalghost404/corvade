'use client';
import { useEffect, useRef, useState } from 'react';

export function useScramble(value: string, duration = 200): string {
  const [display, setDisplay] = useState(value);
  const prevRef = useRef(value);

  useEffect(() => {
    // Only scramble when value actually changes (and not on first render)
    if (prevRef.current === value) return;
    prevRef.current = value;

    const chars = '0123456789.$,';
    let frame = 0;
    const totalFrames = Math.floor(duration / 30); // ~30ms per frame

    const interval = setInterval(() => {
      frame++;
      if (frame >= totalFrames) {
        clearInterval(interval);
        setDisplay(value);
        return;
      }
      // Generate scrambled version: keep non-numeric chars, randomize digits
      const scrambled = value.split('').map(ch => {
        if (/[0-9]/.test(ch)) return chars[Math.floor(Math.random() * 10)]; // random digit
        return ch; // keep $, commas, dots
      }).join('');
      setDisplay(scrambled);
    }, 30);

    return () => clearInterval(interval);
  }, [value, duration]);

  return display;
}
