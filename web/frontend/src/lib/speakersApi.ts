export interface Speaker {
  id: string;
  name: string;
  created_at: number;
}

export const speakersApi = {
  list: async (getAuthHeaders: () => Record<string, string>): Promise<Speaker[]> => {
    const response = await fetch('/api/v1/speakers', {
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
};
