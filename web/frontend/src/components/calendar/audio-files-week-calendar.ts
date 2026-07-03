import './audio-files-week-calendar.css';
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import { formatAudioFileTitle, parseTitleForDate, getSpeakersFromAudioFile } from "@/lib/utils";
import { getSpeakerColorStyles } from "@/lib/speakerColors";

const HOUR_HEIGHT = 60;

interface HourSlot {
  hour: number;
  gapBefore?: number;
}

interface ProcessedAudioFile extends AudioFile {
  dateTime: Date;
  duration: number;
}

export class AudioFilesWeekCalendarElement extends HTMLElement {
  private _data: AudioFile[] = [];
  private _baseDate: Date = new Date();
  private _initialized = false;

  constructor() {
    super();
    this._upgradeProperty('data');
    this._upgradeProperty('baseDate');
  }

  private _upgradeProperty(prop: keyof this) {
    if (Object.prototype.hasOwnProperty.call(this, prop)) {
      const value = this[prop];
      delete this[prop];
      this[prop] = value;
    }
  }

  get data() { return this._data; }
  set data(value: AudioFile[]) {
    this._data = value;
    if (this._initialized) this.render();
  }

  get baseDate() { return this._baseDate; }
  set baseDate(value: Date) {
    this._baseDate = value;
    if (this._initialized) this.render();
  }

  connectedCallback() {
    if (!this._initialized) {
      this._initialized = true;
    }
    this.render();
    this.addEventListener('click', this._handleClick);
    this.addEventListener('mouseover', this._handleMouseOver);
    this.addEventListener('mouseout', this._handleMouseOut);
  }

  disconnectedCallback() {
    this.removeEventListener('click', this._handleClick);
    this.removeEventListener('mouseover', this._handleMouseOver);
    this.removeEventListener('mouseout', this._handleMouseOut);
  }

  private _handleClick = (e: MouseEvent) => {
    const target = e.target as HTMLElement;
    const eventCard = target.closest('.event-card');
    if (eventCard) {
      const fileId = eventCard.getAttribute('data-file-id');
      if (fileId) {
        this.dispatchEvent(new CustomEvent('file-click', {
          detail: { fileId },
          bubbles: true,
          composed: true
        }));
      }
    }

    if (target.closest('.prev-btn')) {
      this.dispatchEvent(new CustomEvent('prev-weeks', { bubbles: true }));
    }

    if (target.closest('.next-btn')) {
      this.dispatchEvent(new CustomEvent('next-weeks', { bubbles: true }));
    }
  }

  private _handleMouseOver = (e: MouseEvent) => {
    const target = e.target as HTMLElement;
    const eventCard = target.closest('.event-card');
    if (eventCard) {
      const fileId = eventCard.getAttribute('data-file-id');
      if (fileId) {
        this.dispatchEvent(new CustomEvent('file-hover-start', {
          detail: { fileId },
          bubbles: true,
          composed: true
        }));
      }
    }
  }

  private _handleMouseOut = (e: MouseEvent) => {
    const target = e.target as HTMLElement;
    const eventCard = target.closest('.event-card');
    if (eventCard) {
      this.dispatchEvent(new CustomEvent('file-hover-end', {
        bubbles: true,
        composed: true
      }));
    }
  }

  private render() {
    const weeksList = this.querySelector('.weeks-list');
    if (!weeksList) return;

    weeksList.innerHTML = '';
    const weeks = this._calculateWeeks();
    const filesWithDateTime = this._processFiles();
    const weekTemplate = (this.querySelector('#week-template') as HTMLTemplateElement);

    if (!weekTemplate) return;

    weeks.forEach(weekStart => {
      const weekNode = weekTemplate.content.cloneNode(true) as DocumentFragment;
      const weekContainer = weekNode.querySelector('.week-container')!;

      const titleElem = weekNode.querySelector('.week-title');
      if (titleElem) {
        titleElem.textContent = `Week of ${weekStart.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}`;
      }

      const { weekFiles } = this._getWeekData(weekStart, filesWithDateTime);
      const { slots, hourToOffset, totalHeight } = this._calculateSlots(weekFiles);

      const daysHeader = weekNode.querySelector('.week-days-header')!;
      const timeColumn = weekNode.querySelector('.time-column')!;
      const daysGrid = weekNode.querySelector('.days-grid')!;

      (timeColumn as HTMLElement).style.height = `${totalHeight}px`;
      (daysGrid as HTMLElement).style.height = `${totalHeight}px`;

      const days = this._getDays(weekStart);

      days.forEach(day => {
        const dayHeader = (this.querySelector('#day-header-template') as HTMLTemplateElement).content.cloneNode(true) as DocumentFragment;
        const columnHeader = dayHeader.querySelector('.day-column-header');
        const isToday = new Date().toDateString() === day.toDateString();
        if (isToday && columnHeader) columnHeader.classList.add('bg-[var(--brand-solid)]/5');

        const dayNameElem = dayHeader.querySelector('.day-name');
        if (dayNameElem) dayNameElem.textContent = day.toLocaleString("default", { weekday: "short" });

        const dayNumberElem = dayHeader.querySelector('.day-number');
        if (dayNumberElem) {
          dayNumberElem.textContent = day.getDate().toString();
          if (isToday) dayNumberElem.classList.add('text-[var(--brand-solid)]');
        }

        daysHeader.appendChild(dayHeader);

        const dayColumn = (this.querySelector('#day-column-template') as HTMLTemplateElement).content.cloneNode(true) as DocumentFragment;
        const columnDiv = dayColumn.querySelector('.day-column')!;

        slots.forEach(slot => {
          const hourCell = (this.querySelector('#hour-cell-template') as HTMLTemplateElement).content.cloneNode(true) as DocumentFragment;
          const cellDiv = hourCell.querySelector('.hour-cell')!;
          if (slot.gapBefore) cellDiv.classList.add('border-t', 'border-t-[var(--brand-solid)]/40');

          this._applyHourStyles(cellDiv, slot.hour);
          columnDiv.appendChild(hourCell);
        });

        weekFiles
          .filter(f => f.dateTime.toDateString() === day.toDateString())
          .forEach(event => {
            const startHour = event.dateTime.getHours();
            if (hourToOffset[startHour] === undefined) return;

            const top = hourToOffset[startHour] + (event.dateTime.getMinutes() / 60) * HOUR_HEIGHT;
            const height = (event.duration / 60) * HOUR_HEIGHT;

            const eventCard = (this.querySelector('#event-card-template') as HTMLTemplateElement).content.cloneNode(true) as DocumentFragment;
            const cardDiv = eventCard.querySelector('.event-card') as HTMLElement;
            cardDiv.setAttribute('data-file-id', event.id);
            cardDiv.style.top = `${top}px`;
            cardDiv.style.height = `${height}px`;

            const eventTitleElem = cardDiv.querySelector('.event-title');
            if (eventTitleElem) {
              eventTitleElem.textContent = event.title ? formatAudioFileTitle(event.title) : `File ${event.id.substring(0, 8)}`;
            }

            // Tooltip population
            const tooltipTitleElem = cardDiv.querySelector('.title-text');
            if (tooltipTitleElem) {
              tooltipTitleElem.textContent = event.title || `Recording ${event.id.substring(0, 8)}`;
            }

            const speakers = getSpeakersFromAudioFile(event);
            const speakersList = cardDiv.querySelector('.speakers-list')!;
            const noSpeakersMsg = cardDiv.querySelector('.no-speakers-msg')!;

            if (speakers.length > 0) {
              speakers.forEach(s => {
                const badgeTemplate = (this.querySelector('#speaker-badge-template') as HTMLTemplateElement).content.cloneNode(true) as DocumentFragment;
                const badge = badgeTemplate.querySelector('.speaker-badge') as HTMLElement;
                badge.textContent = s;
                const styles = getSpeakerColorStyles(s);
                Object.entries(styles).forEach(([k, v]) => badge.style.setProperty(k, v as string));
                speakersList.appendChild(badgeTemplate);
              });
            } else {
              noSpeakersMsg.classList.remove('hidden');
            }

            columnDiv.appendChild(eventCard);
          });

        daysGrid.appendChild(dayColumn);
      });

      slots.forEach(slot => {
        const hourCell = (this.querySelector('#hour-cell-template') as HTMLTemplateElement).content.cloneNode(true) as DocumentFragment;
        const cellDiv = hourCell.querySelector('.hour-cell')!;
        if (slot.gapBefore) {
          cellDiv.classList.add('border-t', 'border-t-[var(--brand-solid)]/40');
          const gapLabel = document.createElement('div');
          gapLabel.className = 'absolute -top-3 left-1 z-20 text-[9px] font-bold text-[var(--brand-solid)] uppercase';
          gapLabel.textContent = `${slot.gapBefore}h`;
          cellDiv.appendChild(gapLabel);
        }
        this._applyHourStyles(cellDiv, slot.hour);
        const hour = slot.hour;
        const label = hour === 0 ? '12 AM' : hour < 12 ? `${hour} AM` : hour === 12 ? '12 PM' : `${hour - 12} PM`;
        cellDiv.textContent = label;
        timeColumn.appendChild(hourCell);
      });

      weeksList.appendChild(weekContainer);
    });
  }

  private _applyHourStyles(element: Element, hour: number) {
    if (hour >= 0 && hour < 6) element.classList.add('night-range');
    else if (hour >= 18) element.classList.add('evening-range');
    else if (hour === 17) element.classList.add('transition-range');
  }

  private _calculateWeeks() {
    const result = [];
    for (let i = 0; i < 4; i++) {
      const date = new Date(this._baseDate);
      date.setDate(this._baseDate.getDate() - (i * 7));
      const day = date.getDay();
      const diff = date.getDate() - day;
      const startOfWeek = new Date(new Date(date.setDate(diff)).setHours(0, 0, 0, 0));
      result.push(startOfWeek);
    }
    return result;
  }

  private _processFiles(): ProcessedAudioFile[] {
    return this._data.map(file => {
      if (file.title) {
        const parsed = parseTitleForDate(file.title);
        if (parsed) return { ...file, dateTime: parsed.date, duration: parsed.duration };
      }
      return { ...file, dateTime: new Date(file.created_at), duration: file.duration || 15 };
    });
  }

  private _getWeekData(startOfWeek: Date, filesWithDateTime: ProcessedAudioFile[]) {
    const endOfWeek = new Date(startOfWeek);
    endOfWeek.setDate(startOfWeek.getDate() + 7);
    const weekFiles = filesWithDateTime.filter(f => f.dateTime >= startOfWeek && f.dateTime < endOfWeek);
    return { endOfWeek, weekFiles };
  }

  private _calculateSlots(weekFiles: ProcessedAudioFile[]) {
    const hourHasActivity = new Array(24).fill(false);
    weekFiles.forEach(file => {
      const startHour = file.dateTime.getHours();
      const endHour = new Date(file.dateTime.getTime() + file.duration * 60000).getHours();
      for (let h = startHour; h <= endHour; h++) if (h < 24) hourHasActivity[h] = true;
    });

    const slots: HourSlot[] = [];
    let currentGap: number[] = [];
    for (let h = 0; h < 24; h++) {
      if (hourHasActivity[h]) {
        if (currentGap.length > 0) {
          if (currentGap.length < 2) currentGap.forEach(gh => slots.push({ hour: gh }));
          else {
            if (slots.length > 0) slots.push({ hour: h, gapBefore: currentGap.length });
            else slots.push({ hour: h });
            currentGap = []; continue;
          }
          currentGap = [];
        }
        slots.push({ hour: h });
      } else currentGap.push(h);
    }
    if (currentGap.length > 0 && currentGap.length < 2) currentGap.forEach(gh => slots.push({ hour: gh }));

    const hourToOffset: Record<number, number> = {};
    slots.forEach((slot, index) => hourToOffset[slot.hour] = index * HOUR_HEIGHT);
    return { slots, hourToOffset, totalHeight: slots.length * HOUR_HEIGHT };
  }

  private _getDays(startOfWeek: Date) {
    const days = [];
    for (let i = 0; i < 7; i++) {
      const date = new Date(startOfWeek);
      date.setDate(startOfWeek.getDate() + i);
      days.push(date);
    }
    return days;
  }
}

if (!customElements.get('audio-files-week-calendar')) {
  customElements.define('audio-files-week-calendar', AudioFilesWeekCalendarElement);
}
