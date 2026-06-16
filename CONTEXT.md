# Cometline domain vocabulary

Terms used across the desktop shell and documented architecture.

## ConversationController

A depth module that owns the full chat turn lifecycle for one session: turn queue serialization, flight/skipUser decision tree, pending-first-message consumption, transcript load gating, and composer phase side effects. `ChatView` is presentation-only and wires flight components through adapters.

## ChatTurn

One user submit through the controller: optional user-bubble flight, SSE stream to CometMind, session title refresh, and queue drain for overlapping submits.
