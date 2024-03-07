// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {PropertyProps} from '../types'
import BaseTextEditor from '../baseTextEditor'

const Number = (props: PropertyProps): JSX.Element => {
    return (
        <BaseTextEditor
            {...props}
            validator={(value) => {
                const valueToValidate = typeof value === 'undefined' ? props.propertyValue as string : value

                return valueToValidate === '' || !isNaN(parseInt(valueToValidate, 10))
            }}
        />
    )
}
export default Number
