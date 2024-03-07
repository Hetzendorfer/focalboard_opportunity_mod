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
                let valueToValidate = value //the current value of the input field
                if (typeof valueToValidate === 'undefined') {
                    valueToValidate = props.propertyValue as string //use the property value, might be diverent from the input field value
                }
                return valueToValidate === '' || !isNaN(parseInt(valueToValidate, 10))
            }}
        />
    )
}
export default Number
