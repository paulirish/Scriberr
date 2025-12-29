export const getSpeakerColorStyles = (speaker: string) => {
    // Simple hash function to get consistent values for a speaker
    let hash = 0;
    for (let i = 0; i < speaker.length; i++) {
        hash = speaker.charCodeAt(i) + ((hash << 5) - hash);
    }

    // Convert hash to hue (0-360)
    const hue = Math.abs(hash % 360);

    // Vary Chroma between 0.02 and 0.06 (subtle but distinct)
    const chroma = 0.02 + (Math.abs((hash >> 8) % 40) / 1000);

    // Vary Lightness slightly for more variety
    // Light mode: 0.90 to 0.95
    const lightL = 0.80 + (Math.abs((hash >> 16) % 150) / 1000);
    // Dark mode: 0.25 to 0.35
    const darkL = 0.25 + (Math.abs((hash >> 24) % 100) / 1000);

    // Base colors
    const baseColor = `oklch(${lightL.toFixed(3)} ${chroma.toFixed(3)} ${hue})`;
    const darkBaseColor = `oklch(${darkL.toFixed(3)} ${chroma.toFixed(3)} ${hue})`;

    // Border colors (darker shades using relative color syntax)
    const borderColor = `oklch(from ${baseColor} calc(l * 0.8) c h)`;
    const darkBorderColor = `oklch(from ${darkBaseColor} calc(l * 0.8) c h)`;

    // Text colors (high contrast)
    const textColor = `oklch(0.3 0.09 ${hue})`;
    const darkTextColor = `oklch(0.85 0.04 ${hue})`;

    return {
        '--speaker-bg': baseColor,
        '--speaker-border': borderColor,
        '--speaker-text': textColor,
        '--speaker-bg-dark': darkBaseColor,
        '--speaker-border-dark': darkBorderColor,
        '--speaker-text-dark': darkTextColor,
    } as React.CSSProperties;
};

/**
 * Returns a Tailwind-compatible class string that uses the CSS variables
 * defined in getSpeakerColorStyles.
 */
export const speakerColorClass = "bg-[var(--speaker-bg)] border-[var(--speaker-border)] text-[var(--speaker-text)] dark:bg-[var(--speaker-bg-dark)] dark:border-[var(--speaker-border-dark)] dark:text-[var(--speaker-text-dark)]";
