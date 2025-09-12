import { h as AuthContext, k as checkPassword } from '../../shared/better-auth.BWDlmABj.cjs';
import 'kysely';
import 'better-call';
import 'zod/v4';
import '../../shared/better-auth.DTtXpZYr.cjs';
import '../../shared/better-auth.BaMSx6K3.cjs';
import 'zod/v4/core';
import 'zod';
import 'better-sqlite3';
import 'bun:sqlite';
import 'node:sqlite';

interface HaveIBeenPwnedOptions {
    customPasswordCompromisedMessage?: string;
}
declare const haveIBeenPwned: (options?: HaveIBeenPwnedOptions) => {
    id: "haveIBeenPwned";
    init(ctx: AuthContext): {
        context: {
            password: {
                hash(password: string): Promise<string>;
                verify: (data: {
                    password: string;
                    hash: string;
                }) => Promise<boolean>;
                config: {
                    minPasswordLength: number;
                    maxPasswordLength: number;
                };
                checkPassword: typeof checkPassword;
            };
        };
    };
    $ERROR_CODES: {
        readonly PASSWORD_COMPROMISED: "The password you entered has been compromised. Please choose a different password.";
    };
};

export { haveIBeenPwned };
export type { HaveIBeenPwnedOptions };
