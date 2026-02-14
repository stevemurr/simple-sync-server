#!/usr/bin/env python3
"""
Simple Sync Server - A lightweight JSON-based sync server for notes and documents.
Provides REST API with last-write-wins conflict resolution.
"""

import json
import os
from datetime import datetime
from pathlib import Path
from typing import Optional

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

app = FastAPI(title="Simple Sync Server")

# Allow all origins for local development
# In production, configure ALLOWED_ORIGINS environment variable
allowed_origins = os.getenv("ALLOWED_ORIGINS", "*").split(",")
app.add_middleware(
    CORSMiddleware,
    allow_origins=allowed_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Data storage path (configurable via environment variable)
DATA_DIR = Path(os.getenv("DATA_DIR", Path(__file__).parent / "data"))
NOTES_FILE = DATA_DIR / "notes.json"


class ChatMessage(BaseModel):
    id: str
    role: str  # "user" or "assistant"
    content: str
    timestamp: str  # ISO format


class Note(BaseModel):
    dateKey: str  # "yyyy-MM-dd" or any unique key
    content: str
    updatedAt: str  # ISO format
    chatMessages: list[ChatMessage] = []
    conversationStarted: bool = False


class SyncRequest(BaseModel):
    notes: list[Note]
    lastSyncTime: Optional[str] = None  # ISO format


class SyncResponse(BaseModel):
    notes: list[Note]
    serverTime: str


def load_notes() -> dict[str, dict]:
    """Load notes from JSON file."""
    if not NOTES_FILE.exists():
        return {}
    try:
        with open(NOTES_FILE, "r") as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError):
        return {}


def save_notes(notes: dict[str, dict]):
    """Save notes to JSON file."""
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    with open(NOTES_FILE, "w") as f:
        json.dump(notes, f, indent=2)


@app.get("/")
def root():
    return {"status": "ok", "service": "Simple Sync Server"}


@app.get("/health")
def health():
    """Health check endpoint for container orchestration."""
    return {"status": "healthy"}


@app.get("/notes", response_model=list[Note])
def get_all_notes():
    """Get all notes."""
    notes = load_notes()
    return list(notes.values())


@app.get("/notes/{date_key}", response_model=Note)
def get_note(date_key: str):
    """Get a specific note by date key."""
    notes = load_notes()
    if date_key not in notes:
        raise HTTPException(status_code=404, detail="Note not found")
    return notes[date_key]


@app.put("/notes/{date_key}", response_model=Note)
def upsert_note(date_key: str, note: Note):
    """Create or update a note."""
    notes = load_notes()

    # Check if we should update (last-write-wins based on updatedAt)
    if date_key in notes:
        existing = notes[date_key]
        existing_time = datetime.fromisoformat(existing["updatedAt"].replace("Z", "+00:00"))
        new_time = datetime.fromisoformat(note.updatedAt.replace("Z", "+00:00"))

        # Only update if the incoming note is newer
        if new_time <= existing_time:
            return Note(**existing)

    notes[date_key] = note.model_dump()
    save_notes(notes)
    return note


@app.delete("/notes/{date_key}")
def delete_note(date_key: str):
    """Delete a note."""
    notes = load_notes()
    if date_key in notes:
        del notes[date_key]
        save_notes(notes)
    return {"status": "deleted", "dateKey": date_key}


@app.post("/sync", response_model=SyncResponse)
def sync_notes(request: SyncRequest):
    """
    Two-way sync endpoint.
    - Receives client notes and merges them (last-write-wins)
    - Returns all server notes that are newer than client's lastSyncTime
    """
    server_notes = load_notes()
    server_time = datetime.utcnow().isoformat() + "Z"

    # Parse client's last sync time
    last_sync = None
    if request.lastSyncTime:
        try:
            last_sync = datetime.fromisoformat(request.lastSyncTime.replace("Z", "+00:00"))
        except ValueError:
            last_sync = None

    # Merge incoming notes (last-write-wins)
    for note in request.notes:
        date_key = note.dateKey
        new_time = datetime.fromisoformat(note.updatedAt.replace("Z", "+00:00"))

        if date_key in server_notes:
            existing_time = datetime.fromisoformat(
                server_notes[date_key]["updatedAt"].replace("Z", "+00:00")
            )
            if new_time > existing_time:
                server_notes[date_key] = note.model_dump()
        else:
            server_notes[date_key] = note.model_dump()

    save_notes(server_notes)

    # Return notes that are newer than client's last sync time
    notes_to_return = []
    for note_data in server_notes.values():
        if last_sync is None:
            # First sync - return everything
            notes_to_return.append(Note(**note_data))
        else:
            note_time = datetime.fromisoformat(
                note_data["updatedAt"].replace("Z", "+00:00")
            )
            if note_time > last_sync:
                notes_to_return.append(Note(**note_data))

    return SyncResponse(notes=notes_to_return, serverTime=server_time)


@app.get("/notes/since/{timestamp}")
def get_notes_since(timestamp: str):
    """Get notes updated since a given timestamp."""
    try:
        since = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
    except ValueError:
        raise HTTPException(status_code=400, detail="Invalid timestamp format")

    notes = load_notes()
    result = []

    for note_data in notes.values():
        note_time = datetime.fromisoformat(
            note_data["updatedAt"].replace("Z", "+00:00")
        )
        if note_time > since:
            result.append(note_data)

    return result


if __name__ == "__main__":
    import uvicorn

    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8080"))
    uvicorn.run(app, host=host, port=port)
