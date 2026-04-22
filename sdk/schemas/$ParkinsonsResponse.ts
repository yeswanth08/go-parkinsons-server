/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $ParkinsonsResponse = {
    properties: {
        isHavingParkinsons: {
            type: 'boolean',
        },
        severity: {
            type: 'number',
            format: 'float',
        },
        suggestion: {
            type: 'string',
        },
        extractedVoiceFeatures: {
            type: 'dictionary',
            contains: {
                properties: {
                },
            },
        },
    },
} as const;
