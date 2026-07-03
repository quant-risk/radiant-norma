-- Migration 002 — adiciona coluna xml_content em envios para armazenar XML completo.
-- Necessário para que cmd/worker possa re-validar e re-submeter envios sem
-- depender de storage externo.

ALTER TABLE envios ADD COLUMN xml_content TEXT NOT NULL DEFAULT '';
ALTER TABLE envios ADD COLUMN zip_content BLOB;