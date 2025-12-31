import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import type {
    AudioFile,
    ExecutionData,
    Transcript
} from "@/types/transcription";

export interface LogsData {
    job_id: string;
    available: boolean;
    content: string;
    message?: string;
}

export function useAudioDetail(audioId: string) {
    const { getAuthHeaders } = useAuth();

    return useQuery({
        queryKey: ["audio", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to fetch audio details");
            return response.json() as Promise<AudioFile>;
        },
        // Poll while processing or pending
        refetchInterval: (query) => {
            const status = query.state.data?.status;
            if (status === "processing" || status === "pending") {
                return 3000;
            }
            return false;
        },
    });
}

export function useTranscript(audioId: string, enabled: boolean) {
    const { getAuthHeaders } = useAuth();

    return useQuery({
        queryKey: ["transcript", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}/transcript`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to fetch transcript");
            const data = await response.json();

            // Handle graceful empty responses (available=false)
            if (data.available === false || !data.transcript) {
                return null; // Return null to indicate no transcript
            }

            // Normalize transcript structure
            if (typeof data.transcript === "string") {
                return { text: data.transcript } as Transcript;
            } else if (data.transcript.text) {
                return {
                    text: data.transcript.text,
                    segments: data.transcript.segments,
                    word_segments: data.transcript.word_segments,
                } as Transcript;
            } else if (data.transcript.segments) {
                const fullText = data.transcript.segments
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    .map((segment: any) => segment.text)
                    .join(" ");
                return {
                    text: fullText,
                    segments: data.transcript.segments,
                    word_segments: data.transcript.word_segments,
                } as Transcript;
            }

            return { text: "" } as Transcript;
        },
        enabled: enabled,
    });
}

export function useExecutionData(audioId: string) {
    const { getAuthHeaders } = useAuth();
    return useQuery({
        queryKey: ["executionData", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}/execution`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to fetch execution data");
            return response.json() as Promise<ExecutionData>;
        },
        enabled: !!audioId,
    });
}

export function useLogs(audioId: string) {
    const { getAuthHeaders } = useAuth();
    return useQuery({
        queryKey: ["logs", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}/logs`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to fetch logs");
            return response.json() as Promise<LogsData>;
        },
        enabled: !!audioId,
    });
}

export function useUpdateTitle(audioId: string) {
    const { getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (newTitle: string) => {
            const response = await fetch(`/api/v1/transcription/${audioId}/title`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    ...getAuthHeaders(),
                },
                body: JSON.stringify({ title: newTitle }),
            });
            if (!response.ok) {
                const msg = await response.text();
                throw new Error(msg || "Failed to update title");
            }
            return response.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["audio", audioId] });
            queryClient.invalidateQueries({ queryKey: ["audioFiles"] }); // Update list too
        },
    });
}
