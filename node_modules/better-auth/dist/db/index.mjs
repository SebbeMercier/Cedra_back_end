export { d as convertFromDB, b as convertToDB, c as createInternalAdapter, a as getAdapter, e as getMigrations, f as getSchema, g as getWithHooks, m as matchType } from '../shared/better-auth.DV0Lly8-.mjs';
export { g as getAuthTables } from '../shared/better-auth.B_PTVODM.mjs';
export { t as toZodSchema } from '../shared/better-auth.DrJWSFx6.mjs';
export { a as accountSchema, c as coreSchema, g as getAllFields, m as mergeSchema, j as parseAccountInput, d as parseAccountOutput, i as parseAdditionalUserInput, f as parseInputData, p as parseOutputData, k as parseSessionInput, e as parseSessionOutput, h as parseUserInput, b as parseUserOutput, s as sessionSchema, u as userSchema, v as verificationSchema } from '../shared/better-auth.Dcv8PS7T.mjs';
import '../shared/better-auth.CW6D9eSx.mjs';
import '../shared/better-auth.O2VtDkDK.mjs';
import '../shared/better-auth.CMQ3rA-I.mjs';
import '../shared/better-auth.BZZKN1g7.mjs';
import '../shared/better-auth.BjBlybv-.mjs';
import '../shared/better-auth.BUPPRXfK.mjs';
import '@better-auth/utils/random';
import 'zod/v4';
import 'better-call';
import '@better-auth/utils/hash';
import '@noble/ciphers/chacha.js';
import '@noble/ciphers/utils.js';
import '@better-auth/utils/base64';
import 'jose';
import '@noble/hashes/scrypt.js';
import '@better-auth/utils';
import '@better-auth/utils/hex';
import '@noble/hashes/utils.js';
import '../shared/better-auth.B4Qoxdgc.mjs';
import '../shared/better-auth.DdzSJf-n.mjs';
import '../shared/better-auth.BduPtJy8.mjs';
import 'kysely';
import '../shared/better-auth.DOXaXTC6.mjs';
import '../shared/better-auth.7ZeACU2v.mjs';

const createFieldAttribute = (type, config) => {
  return {
    type,
    ...config
  };
};

export { createFieldAttribute };
