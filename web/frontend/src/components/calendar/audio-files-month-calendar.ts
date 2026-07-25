import './audio-files-month-calendar.css';
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import { formatAudioFileTitle, parseTitleForDate, getSpeakersFromAudioFile } from "@/lib/utils";
import { getSpeakerColorStyles } from "@/lib/speakerColors";

const daysOfWeek = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export class AudioFilesMonthCalendarElement extends HTMLElement {
  private _data: AudioFile[] = [];
  private _currentDate: Date = new Date();
  private _initialized = false;

  constructor() {
    super();
    this._upgradeProperty('data');
    this._upgradeProperty('currentDate');
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

  get currentDate() { return this._currentDate; }
  set currentDate(value: Date) {
    this._currentDate = value;
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
      this.dispatchEvent(new CustomEvent('prev-month', { bubbles: true }));
    }

    if (target.closest('.next-btn')) {
      this.dispatchEvent(new CustomEvent('next-month', { bubbles: true }));
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
    const gridContainer = this.querySelector('.grid-container');
    const monthTitle = this.querySelector('.month-title');
    if (!gridContainer || !monthTitle) return;

    gridContainer.innerHTML = '';
    monthTitle.textContent = this._currentDate.toLocaleString("default", { month: "long", year: "numeric" });

    daysOfWeek.forEach(day => {
      const dayNameTemplate = (this.querySelector('#day-name-template') as HTMLTemplateElement);
      if (dayNameTemplate) {
        const node = dayNameTemplate.content.cloneNode(true) as DocumentFragment;
        const nameElem = node.querySelector('.day-name');
        if (nameElem) nameElem.textContent = day;
        gridContainer.appendChild(node);
      }
    });

    const firstDayOfMonth = new Date(this._currentDate.getFullYear(), this._currentDate.getMonth(), 1);
    const lastDayOfMonth = new Date(this._currentDate.getFullYear(), this._currentDate.getMonth() + 1, 0);
    const startingDay = firstDayOfMonth.getDay();
    const totalDays = lastDayOfMonth.getDate();

    const filesByDate = this._groupFilesByDate();

    for (let i = 0; i < startingDay; i++) {
      const emptyTemplate = (this.querySelector('#empty-cell-template') as HTMLTemplateElement);
      if (emptyTemplate) {
        gridContainer.appendChild(emptyTemplate.content.cloneNode(true));
      }
    }

    for (let day = 1; day <= totalDays; day++) {
      const date = new Date(this._currentDate.getFullYear(), this._currentDate.getMonth(), day);
      const dateString = date.toDateString();
      const filesForDay = filesByDate[dateString] || [];
      const isToday = new Date().toDateString() === dateString;

      const cellTemplate = (this.querySelector('#day-cell-template') as HTMLTemplateElement);
      if (cellTemplate) {
        const node = cellTemplate.content.cloneNode(true) as DocumentFragment;
        const cellDiv = node.querySelector('.day-cell');
        if (cellDiv && isToday) cellDiv.classList.add('today');

        const numSpan = node.querySelector('.day-number');
        if (numSpan) {
          numSpan.textContent = day.toString();
          if (isToday) numSpan.classList.add('text-[var(--brand-solid)]');
        }

        const eventsContainer = node.querySelector('.events-container')!;
        filesForDay.forEach(file => {
          const eventTemplate = (this.querySelector('#event-card-template') as HTMLTemplateElement);
          if (eventTemplate) {
            const eventNode = eventTemplate.content.cloneNode(true) as DocumentFragment;
            const card = eventNode.querySelector('.event-card') as HTMLElement;
            card.setAttribute('data-file-id', file.id);

            const titleElem = eventNode.querySelector('.event-title');
            if (titleElem) {
              titleElem.textContent = file.title ? formatAudioFileTitle(file.title) : `File ${file.id.substring(0, 8)}`;
            }

            // Tooltip population
            const tooltipTitleElem = card.querySelector('.title-text');
            if (tooltipTitleElem) {
              tooltipTitleElem.textContent = file.title || `Recording ${file.id.substring(0, 8)}`;
            }

            const speakers = getSpeakersFromAudioFile(file);
            const speakersList = card.querySelector('.speakers-list')!;
            const noSpeakersMsg = card.querySelector('.no-speakers-msg')!;

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

            eventsContainer.appendChild(eventNode);
          }
        });

        gridContainer.appendChild(node);
      }
    }
  }

  private _groupFilesByDate() {
    return this._data.reduce((acc, file) => {
      let date: Date;
      if (file.title) {
        const parsed = parseTitleForDate(file.title);
        date = parsed ? parsed.date : new Date(file.created_at);
      } else {
        date = new Date(file.created_at);
      }
      const dateString = date.toDateString();
      if (!acc[dateString]) acc[dateString] = [];
      acc[dateString].push(file);
      return acc;
    }, {} as Record<string, AudioFile[]>);
  }
}

if (!customElements.get('audio-files-month-calendar')) {
  customElements.define('audio-files-month-calendar', AudioFilesMonthCalendarElement);
}
