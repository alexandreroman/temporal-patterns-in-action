// AES-256-GCM PayloadCodec — byte-compatible with workers/encryption/codec.go.
// Marshals the full Payload (metadata + data) as protobuf, seals with AES-GCM,
// stores `nonce || ciphertext || authTag` in the outer payload's data field.
// The protobuf encoder is hand-rolled because @temporalio/proto is a transitive
// dep that pnpm does not hoist — importing it from app code is fragile.

import { createCipheriv, createDecipheriv, randomBytes } from "node:crypto";
import type { Payload } from "@temporalio/client";

export const DEMO_KEY = Buffer.from("temporal-demo-encryption-key-32b", "utf8");

const META_ENCODING = "binary/encrypted";
const META_CIPHER = "AES-256-GCM";
const META_KEY_ID = "v1-demo";

const NONCE_BYTES = 12;
const AUTH_TAG_BYTES = 16;

// Declared locally — @temporalio/proto is not hoisted by pnpm, so importing
// it from app code is fragile across installs.
export interface PayloadCodec {
  encode(payloads: Payload[]): Promise<Payload[]>;
  decode(payloads: Payload[]): Promise<Payload[]>;
}

export class EncryptionCodec implements PayloadCodec {
  constructor(private readonly key: Buffer) {}

  async encode(payloads: Payload[]): Promise<Payload[]> {
    return payloads.map((p) => {
      const plaintext = encodePayload(p);
      const nonce = randomBytes(NONCE_BYTES);
      const cipher = createCipheriv("aes-256-gcm", this.key, nonce);
      const ciphertext = Buffer.concat([cipher.update(plaintext), cipher.final()]);
      const authTag = cipher.getAuthTag();
      // Mirror Go's gcm.Seal layout (ciphertext || authTag) so wire bytes match
      // across SDKs.
      return {
        metadata: {
          encoding: Buffer.from(META_ENCODING, "utf8"),
          "encryption-cipher": Buffer.from(META_CIPHER, "utf8"),
          "encryption-key-id": Buffer.from(META_KEY_ID, "utf8"),
        },
        data: Buffer.concat([nonce, ciphertext, authTag]),
      };
    });
  }

  async decode(payloads: Payload[]): Promise<Payload[]> {
    return payloads.map((p) => {
      const encoding = p.metadata?.encoding;
      if (!encoding || bytesToString(encoding) !== META_ENCODING) return p;

      const data = Buffer.from(p.data ?? []);
      if (data.length < NONCE_BYTES + AUTH_TAG_BYTES) {
        throw new Error("encryption-codec: payload too short for nonce+tag");
      }
      const nonce = data.subarray(0, NONCE_BYTES);
      const authTag = data.subarray(data.length - AUTH_TAG_BYTES);
      const ciphertext = data.subarray(NONCE_BYTES, data.length - AUTH_TAG_BYTES);

      const decipher = createDecipheriv("aes-256-gcm", this.key, nonce);
      decipher.setAuthTag(authTag);
      const plaintext = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
      return decodePayload(plaintext);
    });
  }
}

// ---------- Minimal Payload protobuf encoder/decoder ----------
// Proto schema (temporal.api.common.v1.Payload):
//   map<string, bytes> metadata = 1;
//   bytes data = 2;
// Which wire-level expands to:
//   field 1: repeated MetadataEntry { string key = 1; bytes value = 2; }
//   field 2: bytes data

function encodePayload(p: Payload): Buffer {
  const chunks: Buffer[] = [];
  const metadata = p.metadata ?? {};
  for (const [key, value] of Object.entries(metadata)) {
    if (value === null || value === undefined) continue;
    chunks.push(encodeMetadataEntry(key, Buffer.from(value)));
  }
  if (p.data && p.data.length > 0) {
    chunks.push(encodeTag(2, WIRE_LEN));
    chunks.push(encodeVarint(p.data.length));
    chunks.push(Buffer.from(p.data));
  }
  return Buffer.concat(chunks);
}

function encodeMetadataEntry(key: string, value: Buffer): Buffer {
  const inner: Buffer[] = [];
  // key (field 1, string)
  const keyBytes = Buffer.from(key, "utf8");
  inner.push(encodeTag(1, WIRE_LEN), encodeVarint(keyBytes.length), keyBytes);
  // value (field 2, bytes)
  inner.push(encodeTag(2, WIRE_LEN), encodeVarint(value.length), value);
  const innerBuf = Buffer.concat(inner);
  // outer: field 1 of Payload, message
  return Buffer.concat([encodeTag(1, WIRE_LEN), encodeVarint(innerBuf.length), innerBuf]);
}

function decodePayload(buf: Buffer): Payload {
  const metadata: Record<string, Uint8Array> = {};
  let data: Uint8Array | undefined;
  let offset = 0;
  while (offset < buf.length) {
    const tag = readVarint(buf, offset);
    offset = tag.next;
    const field = tag.value >>> 3;
    const wire = tag.value & 0x7;
    if (wire !== WIRE_LEN) {
      offset = skipField(buf, offset, wire);
      continue;
    }
    const len = readVarint(buf, offset);
    offset = len.next;
    const end = offset + len.value;
    if (field === 1) {
      const entry = decodeMetadataEntry(buf.subarray(offset, end));
      if (entry) metadata[entry.key] = entry.value;
    } else if (field === 2) {
      data = Buffer.from(buf.subarray(offset, end));
    }
    offset = end;
  }
  return { metadata, data: data ?? new Uint8Array(0) };
}

function decodeMetadataEntry(buf: Buffer): { key: string; value: Uint8Array } | null {
  let key = "";
  let value: Uint8Array = new Uint8Array(0);
  let offset = 0;
  while (offset < buf.length) {
    const tag = readVarint(buf, offset);
    offset = tag.next;
    const field = tag.value >>> 3;
    const wire = tag.value & 0x7;
    if (wire !== WIRE_LEN) {
      offset = skipField(buf, offset, wire);
      continue;
    }
    const len = readVarint(buf, offset);
    offset = len.next;
    const end = offset + len.value;
    if (field === 1) key = buf.subarray(offset, end).toString("utf8");
    else if (field === 2) value = Buffer.from(buf.subarray(offset, end));
    offset = end;
  }
  return key ? { key, value } : null;
}

// ---------- Varint / wire-type helpers ----------

const WIRE_VARINT = 0;
const WIRE_64 = 1;
const WIRE_LEN = 2;
const WIRE_32 = 5;

function encodeTag(field: number, wire: number): Buffer {
  return encodeVarint((field << 3) | wire);
}

function encodeVarint(value: number): Buffer {
  const out: number[] = [];
  let v = value;
  while (v > 0x7f) {
    out.push((v & 0x7f) | 0x80);
    v = Math.floor(v / 128);
  }
  out.push(v & 0x7f);
  return Buffer.from(out);
}

function readVarint(buf: Buffer, start: number): { value: number; next: number } {
  let value = 0;
  let shift = 0;
  let offset = start;
  while (offset < buf.length) {
    const byte = buf[offset++] ?? 0;
    value += (byte & 0x7f) * Math.pow(2, shift);
    if ((byte & 0x80) === 0) return { value, next: offset };
    shift += 7;
    if (shift > 49) throw new Error("encryption-codec: varint too long");
  }
  throw new Error("encryption-codec: truncated varint");
}

function skipField(buf: Buffer, start: number, wire: number): number {
  switch (wire) {
    case WIRE_VARINT:
      return readVarint(buf, start).next;
    case WIRE_64:
      return start + 8;
    case WIRE_LEN: {
      const len = readVarint(buf, start);
      return len.next + len.value;
    }
    case WIRE_32:
      return start + 4;
    default:
      throw new Error(`encryption-codec: unsupported wire type ${wire}`);
  }
}

function bytesToString(b: Uint8Array): string {
  return Buffer.from(b).toString("utf8");
}
