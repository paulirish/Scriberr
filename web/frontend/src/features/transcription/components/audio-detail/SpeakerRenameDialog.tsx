import React, { useState, useEffect, useCallback } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Loader2, Users, Save, X, Globe } from 'lucide-react';
import { useAuth } from "@/features/auth/hooks/useAuth";
import { speakersApi } from '../lib/speakersApi';
// Note: Install framer-motion for enhanced animations
// import { motion, AnimatePresence } from 'framer-motion';

interface SpeakerMapping {
  id?: number;
  original_speaker: string;
  custom_name: string;
}

interface SpeakerRenameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  transcriptionId: string;
  onSpeakerMappingsUpdate: (mappings: SpeakerMapping[]) => void;
  initialSpeakers?: string[]; // Detected speakers from transcript
}

const SpeakerRenameDialog: React.FC<SpeakerRenameDialogProps> = ({
  open,
  onOpenChange,
  transcriptionId,
  onSpeakerMappingsUpdate,
  initialSpeakers = [],
}) => {
  const { getAuthHeaders } = useAuth();
  const [speakerMappings, setSpeakerMappings] = useState<Record<string, string>>({});
  const [globalUpdates, setGlobalUpdates] = useState<Record<string, boolean>>({});
  const [globalSpeakers, setGlobalSpeakers] = useState<Record<string, boolean>>({}); // Map of names/IDs that are global speakers
  const [globalSpeakerIds, setGlobalSpeakerIds] = useState<Record<string, string>>({}); // Map Name -> UUID
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Initialize speaker mappings when dialog opens
  useEffect(() => {
    if (open && transcriptionId) {
      fetchSpeakerMappings();
      checkGlobalSpeakers();
    }
  }, [open, transcriptionId]);

  const checkGlobalSpeakers = async () => {
    try {
      const speakers = await speakersApi.list(getAuthHeaders);
      const speakerMap: Record<string, boolean> = {};
      const nameToId: Record<string, string> = {};

      speakers.forEach(s => {
        // We match by ID (assuming TitaNet returns Global IDs in the transcript)
        // or potentially by name if we want to allow claiming
        speakerMap[s.id] = true;
        // Also map name if they match (Transcript usually contains the Name)
        speakerMap[s.name] = true;

        // Store mapping from Name -> UUID for renaming
        nameToId[s.name] = s.id;
      });

      setGlobalSpeakers(speakerMap);
      setGlobalSpeakerIds(nameToId);
    } catch (err) {
      console.warn("Failed to check global speakers", err);
    }
  };

  const fetchSpeakerMappings = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/transcription/${transcriptionId}/speakers`, {
        headers: { ...getAuthHeaders() },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch speaker mappings: ${response.statusText}`);
      }

      const existingMappings: SpeakerMapping[] = await response.json();

      // Create a mapping object from the response
      const mappingObj: Record<string, string> = {};

      // Initialize with existing mappings
      existingMappings.forEach(mapping => {
        mappingObj[mapping.original_speaker] = mapping.custom_name;
      });

      // Add any speakers from the transcript that don't have mappings yet
      initialSpeakers.forEach(speaker => {
        if (!mappingObj[speaker]) {
          mappingObj[speaker] = speaker; // Default to original name
        }
      });

      setSpeakerMappings(mappingObj);
    } catch (err) {
      console.error('Error fetching speaker mappings:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch speaker mappings');

      // Initialize with default mappings if fetch fails
      const defaultMappings: Record<string, string> = {};
      initialSpeakers.forEach(speaker => {
        defaultMappings[speaker] = speaker;
      });
      setSpeakerMappings(defaultMappings);
    } finally {
      setIsLoading(false);
    }
  }, [transcriptionId, getAuthHeaders, initialSpeakers]);

  // Initialize speaker mappings when dialog opens
  useEffect(() => {
    if (open && transcriptionId) {
      fetchSpeakerMappings();
    }
  }, [open, transcriptionId, fetchSpeakerMappings]);

  const handleSpeakerNameChange = (originalSpeaker: string, customName: string) => {
    setSpeakerMappings(prev => ({
      ...prev,
      [originalSpeaker]: customName,
    }));
  };

  const handleGlobalUpdateChange = (originalSpeaker: string, checked: boolean) => {
    setGlobalUpdates(prev => ({
      ...prev,
      [originalSpeaker]: checked
    }));
  };

  const saveSpeakerMappings = async () => {
    setIsSaving(true);
    setError(null);

    try {
      // 1. Process Global Updates first
      const globalUpdatePromises = Object.keys(globalUpdates)
        .filter(key => globalUpdates[key] && speakerMappings[key])
        .map(async (key) => {
           // key is the 'originalSpeaker' from the transcript (e.g. "Speaker-123" or "John")
           // We need to find the UUID for this speaker to update Qdrant
           const uuid = globalSpeakerIds[key];

           if (uuid) {
             // Rename using UUID
             await speakersApi.rename(uuid, speakerMappings[key], getAuthHeaders);
           } else {
             console.warn(`Could not find global ID for speaker ${key}`);
             // If we can't find ID, we can't rename globally via API
           }
        });

      if (globalUpdatePromises.length > 0) {
        await Promise.all(globalUpdatePromises);
      }

      // 2. Save Local Mappings (as before)
      // Convert mappings to API format
      const mappingsArray = Object.entries(speakerMappings).map(([original_speaker, custom_name]) => ({
        original_speaker,
        custom_name,
      }));

      const response = await fetch(`/api/v1/transcription/${transcriptionId}/speakers`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify({
          mappings: mappingsArray,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to save speaker mappings: ${response.statusText}`);
      }

      const updatedMappings: SpeakerMapping[] = await response.json();
      onSpeakerMappingsUpdate(updatedMappings);
      onOpenChange(false);
    } catch (err) {
      console.error('Error saving speaker mappings:', err);
      setError(err instanceof Error ? err.message : 'Failed to save speaker mappings');
    } finally {
      setIsSaving(false);
    }
  };

  const speakers = Object.keys(speakerMappings).sort();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            Rename Speakers
          </DialogTitle>
          <DialogDescription>
            Rename speakers locally for this transcript, or update their global identity.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin" />
            <span className="ml-2 text-sm text-muted-foreground">Loading speakers...</span>
          </div>
        ) : (
          <div className="space-y-4">
            {error && (
              <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
              </div>
            )}

            {speakers.length === 0 ? (
              <Card>
                <CardContent className="pt-6 text-center text-muted-foreground">
                  <Users className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>No speakers found with diarization enabled.</p>
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
                {speakers.map((speaker) => (
                  <div
                    key={speaker}
                    className="flex flex-col gap-2 p-3 border rounded-lg bg-card"
                  >
                    <div className="flex items-center justify-between">
                        <Label htmlFor={`speaker-${speaker}`} className="text-sm font-medium">
                        Original: <span className="text-muted-foreground">{speaker}</span>
                        </Label>
                        {/* Show indicator if this is a global speaker */}
                        {(speaker.startsWith("Speaker-") || globalSpeakers[speaker]) && (
                            <span className="text-xs bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-100 px-2 py-0.5 rounded-full flex items-center gap-1">
                                <Globe className="h-3 w-3" /> Global Identity
                            </span>
                        )}
                    </div>

                    <div className="flex gap-3 items-end">
                        <div className="flex-1">
                            <Input
                                id={`speaker-${speaker}`}
                                value={speakerMappings[speaker] || ''}
                                onChange={(e) => handleSpeakerNameChange(speaker, e.target.value)}
                                placeholder={`Enter custom name for ${speaker}`}
                                className="transition-all duration-200 focus:ring-2 focus:ring-primary/20"
                            />
                        </div>
                    </div>

                    {/* Switch to update global profile */}
                    {(speaker.startsWith("Speaker-") || globalSpeakers[speaker]) && (
                        <div className="flex items-center space-x-2 mt-1">
                            <Switch
                                id={`global-${speaker}`}
                                checked={globalUpdates[speaker] || false}
                                onCheckedChange={(checked: boolean) => handleGlobalUpdateChange(speaker, checked)}
                            />
                            <Label
                                htmlFor={`global-${speaker}`}
                                className="text-xs text-muted-foreground cursor-pointer select-none"
                            >
                                Also rename globally (updates all future recognitions)
                            </Label>
                        </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        <DialogFooter className="gap-2 sm:justify-between">
          <div className="text-xs text-muted-foreground self-center hidden sm:block">
            {Object.keys(globalUpdates).filter(k => globalUpdates[k]).length > 0 &&
                `Updating ${Object.keys(globalUpdates).filter(k => globalUpdates[k]).length} global profiles`}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isSaving}>
                <X className="h-4 w-4 mr-1" />
                Cancel
            </Button>
            <Button
                onClick={saveSpeakerMappings}
                disabled={isSaving || speakers.length === 0}
                className="min-w-[100px]"
            >
                {isSaving ? (
                <>
                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                    Saving...
                </>
                ) : (
                <>
                    <Save className="h-4 w-4 mr-1" />
                    Save
                </>
                )}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default SpeakerRenameDialog;
