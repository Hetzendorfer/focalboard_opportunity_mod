// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import { PropertyProps } from '../types'
import BaseTextEditor from '../baseTextEditor'

const Number = (props: PropertyProps): JSX.Element => {
    return (
        <BaseTextEditor
            {...props}
            validator={() => {
                const value = (props.propertyValue as string).trim().replace('€', '')
                let isValid = true
                isValid = props.propertyValue === ''
                isValid = !isNaN(parseInt(value, 10))
                return isValid
            }}
        />
    )
}
export default Number
