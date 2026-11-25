import { describe, expect, it } from 'vitest';
import { formatDuration, formatPriceCents, isSafeImageUrl, parsePositiveId } from './format';

describe('format helpers', () => {
  it('formats cents as currency', () => {
    expect(formatPriceCents(0)).toContain('0.00');
    expect(formatPriceCents(8500)).toContain('85.00');
  });

  it('formats movie durations cleanly', () => {
    expect(formatDuration(45)).toBe('45 min');
    expect(formatDuration(120)).toBe('2 h');
    expect(formatDuration(166)).toBe('2 h 46 min');
  });

  it('parses only positive integer route IDs', () => {
    expect(parsePositiveId('1')).toBe(1);
    expect(parsePositiveId('42')).toBe(42);
    expect(parsePositiveId('0')).toBeNull();
    expect(parsePositiveId('-5')).toBeNull();
    expect(parsePositiveId('abc')).toBeNull();
    expect(parsePositiveId('12abc')).toBeNull();
    expect(parsePositiveId(undefined)).toBeNull();
  });

  it('accepts only http/https poster URLs', () => {
    expect(isSafeImageUrl('https://cdn.example.com/dune.jpg')).toBe(true);
    expect(isSafeImageUrl('http://localhost:8080/poster.png')).toBe(true);
    expect(isSafeImageUrl('')).toBe(false);
    expect(isSafeImageUrl('   ')).toBe(false);
    expect(isSafeImageUrl('javascript:alert(1)')).toBe(false);
    expect(isSafeImageUrl('data:text/html,<script>')).toBe(false);
  });
});
