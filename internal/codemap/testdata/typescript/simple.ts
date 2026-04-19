import { readFileSync } from 'fs';
import { helper } from './local';
import express from 'express';

export function greet(name: string): string {
    return `Hello, ${name}!`;
}

export class MyClass {
    private value: number;

    constructor(value: number) {
        this.value = value;
    }

    getValue(): number {
        return this.value;
    }
}

export interface Handler {
    handle(input: string): void;
}

export type Config = {
    host: string;
    port: number;
};

export enum Status {
    Active = 'active',
    Inactive = 'inactive',
}
