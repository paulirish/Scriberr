/// <reference types="vite/client" />

import * as React from 'react';

declare global {
  namespace JSX {
    interface IntrinsicElements {
      'audio-files-week-calendar': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>;
      'audio-files-month-calendar': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>;
    }
  }
}
