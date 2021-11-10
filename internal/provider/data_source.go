package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jmespath/go-jmespath"
)

func dataSource() *schema.Resource {
	return &schema.Resource{
		Description: "The `jmespath` data source allows an jmespath program implementing a specific protocol " +
			"(defined below) to act as a data source, exposing arbitrary data for use elsewhere in the Terraform " +
			"configuration.\n" +
			"\n" +
			"**Warning** This mechanism is provided as an \"escape hatch\" for exceptional situations where a " +
			"first-class Terraform provider is not more appropriate. Its capabilities are limited in comparison " +
			"to a true data source, and implementing a data source via an jmespath program is likely to hurt the " +
			"portability of your Terraform configuration by creating dependencies on jmespath programs and " +
			"libraries that may not be available (or may need to be used differently) on different operating " +
			"systems.\n" +
			"\n" +
			"**Warning** Terraform Enterprise does not guarantee availability of any particular language runtimes " +
			"or jmespath programs beyond standard shell utilities, so it is not recommended to use this data source " +
			"within configurations that are applied within Terraform Enterprise.",

		Read: dataSourceRead,

		Schema: map[string]*schema.Schema{
			"program": {
				Description: "A list of strings, whose first element is the program to run and whose " +
					"subsequent elements are optional command line arguments to the program. Terraform does " +
					"not execute the program through a shell, so it is not necessary to escape shell " +
					"metacharacters nor add quotes around arguments containing spaces.",
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"working_dir": {
				Description: "Working directory of the program. If not supplied, the program will run " +
					"in the current directory.",
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"query": {
				Description: "A map of string values to pass to the jmespath program as the query " +
					"arguments. If not supplied, the program will receive an empty object as its input.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"result": {
				Description: "A map of string values returned from the jmespath program.",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceRead(d *schema.ResourceData, meta interface{}) error {

	programI := d.Get("program").([]interface{})
	workingDir := d.Get("working_dir").(string)
	query := d.Get("query").(map[string]interface{})

	// This would be a ValidateFunc if helper/schema allowed these
	// to be applied to lists.
	if err := validateProgramAttr(programI); err != nil {
		return err
	}

	program := make([]string, len(programI))
	for i, vI := range programI {
		program[i] = vI.(string)
	}

	cmd := exec.Command(program[0], program[1:]...)

	cmd.Dir = workingDir

	resultJson, err := cmd.Output()
	log.Printf("[TRACE] JSON output: %+v\n", string(resultJson))
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Stderr != nil && len(exitErr.Stderr) > 0 {
				return fmt.Errorf("failed to execute %q: %s", program[0], string(exitErr.Stderr))
			}
			return fmt.Errorf("command %q failed with no error message", program[0])
		} else {
			return fmt.Errorf("failed to execute %q: %s", program[0], err)
		}
	}

	result := make(map[string]string)

	var data interface{}

	for k, v := range query {
		err := json.Unmarshal(resultJson, &data)
		if err != nil {
			return fmt.Errorf("command %q produced invalid JSON: %s", program[0], err)
		}
		searchResult, err := jmespath.Search(v.(string), data)
		if err != nil {
			return fmt.Errorf("error jmespath.Search: %s\n", err)
		} else {
			switch searchResult.(type) {
			case int:
				result[k] = strconv.Itoa(searchResult.(int))
			case float64:
				result[k] = strconv.FormatFloat(searchResult.(float64), 'f', 2, 32)
			case string:
				result[k] = searchResult.(string)
			case nil:
				log.Printf("[INFO] json value not find for: %s\n", v.(string))
			default:
				log.Printf("[INFO] json value type not implemented: %s\n", v.(string))
			}

		}
	}

	d.Set("result", result)

	d.SetId("-")
	return nil
}
