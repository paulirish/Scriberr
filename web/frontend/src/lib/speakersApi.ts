export interface Speaker {
  id: string;
  name: string;
  created_at: number;
}

export interface Speaker {
  id: string;
  name: string;
  created_at: number;
}

export interface SpeakerSegment {
  id: number;
  transcription_job_id: string;
  speaker_id: string;
  start: number;
  end: number;
  text: string;
  created_at: string;
}

export const speakersApi = {
  list: async (getAuthHeaders: () => Record<string, string>): Promise<Speaker[]> => {
    const response = await fetch('/api/v1/speakers/', {
      headers: getAuthHeaders(),
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || 'Failed to fetch speakers');
    }
    return response.json();
  },

  rename: async (id: string, name: string, getAuthHeaders: () => Record<string, string>): Promise<void> => {
    const response = await fetch(`/api/v1/speakers/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      },
      body: JSON.stringify({ name }),
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || 'Failed to rename speaker');
    }
  },

  delete: async (id: string, getAuthHeaders: () => Record<string, string>): Promise<void> => {
    const response = await fetch(`/api/v1/speakers/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || 'Failed to delete speaker');
    }
  },

  getSegments: async (id: string, getAuthHeaders: () => Record<string, string>): Promise<SpeakerSegment[]> => {
    const response = await fetch(`/api/v1/speakers/${id}/segments`, {
      headers: getAuthHeaders(),
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || 'Failed to fetch speaker segments');
    }
    return response.json();
  },

  getSegmentAudioUrl: (speakerId: string, segmentId: number): string => {
    return `/api/v1/speakers/${speakerId}/segments/${segmentId}/audio`;
  }
};
