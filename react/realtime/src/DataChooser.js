import * as React from 'react';
import { RadioGroup, RadioButton } from 'react-radio-buttons'; 
import './ChooseActivity.css'
import DataViewer from './DataViewer.js'
import DataViewer2 from './DataViewer2.js'
import DataViewer3 from './DataViewer3.js'
import DataViewer4 from './DataViewer4.js'


class DataChooser extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
          value: "none",
       };
    }

    onChange(val) {
      console.log(val);
            this.setState({value:val});
    }

    
    render() {

      return (
          <div>
            <h2 style={ { marginTop: 16 } }>Choose data:</h2>

            <RadioGroup className="activities" onChange={ this.onChange.bind(this) } value={this.state.value} horizontal>
          <RadioButton value="basic">
          basic
          </RadioButton>
          <RadioButton value="accelerometer">
          accelerometer
          </RadioButton>
          <RadioButton value="gyroscope">
          gyroscope
          </RadioButton>
          <RadioButton value="magnetometer">
          magnetometer
          </RadioButton>
        </RadioGroup>
        {(() => {
        switch (this.state.value) {
          case "accelerometer":   return <DataViewer2 />;
          case "gyroscope": return <DataViewer3 />;
          case "magnetometer": return <DataViewer4 />;
          default:      return <DataViewer />;
        }
      })()}

          </div>
        );
      }
}

export default DataChooser;